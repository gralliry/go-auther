// Package json provides a JSON file-backed adapter for Auther.
//
// Each adapter exclusively owns its file via a lock file. A background
// goroutine updates the lock timestamp every second. If the owning process
// crashes, the stale lock is automatically cleaned up on the next attempt
// to open the file (5 second threshold).
//
// Mutations update in-memory data only; the background goroutine flushes
// to disk every second. Close() ensures any remaining dirty data is written.
package json

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	json "github.com/json-iterator/go"
	"github.com/gralliry/go-auther/adapter"
)

const (
	flushInterval  = time.Second
	lockStaleAfter = 5 * time.Second
)

// Adapter is a JSON file-backed, mutex-protected store.
type Adapter struct {
	mu       sync.RWMutex
	file     *os.File
	lock     *os.File // lock file, held for lifetime
	lockPath string
	data     adapter.Snapshot
	dirty    bool // true when in-memory data differs from disk

	flushStop chan struct{} // close to stop the flush goroutine
}

// New opens (or creates) the JSON file, acquires an exclusive lock, loads the
// snapshot into memory, and starts the periodic flush goroutine.
func New(path string) (*Adapter, error) {
	lockPath := path + ".lock"
	lk, err := acquireLock(lockPath)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		lk.Close()
		os.Remove(lockPath)
		return nil, err
	}

	raw, err := io.ReadAll(f)
	if err != nil {
		f.Close()
		lk.Close()
		os.Remove(lockPath)
		return nil, err
	}

	a := &Adapter{file: f, lock: lk, lockPath: lockPath, flushStop: make(chan struct{})}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &a.data); err != nil {
			f.Close()
			lk.Close()
			os.Remove(lockPath)
			return nil, err
		}
	}
	go a.flushLoop()
	return a, nil
}

// acquireLock creates the lock file exclusively. If the lock is stale (older
// than lockStaleAfter), it is automatically cleaned up.
func acquireLock(path string) (*os.File, error) {
	lk, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err == nil {
		return lk, nil
	}

	// Lock exists — check if stale.
	fi, err := os.Stat(path)
	if err == nil && time.Since(fi.ModTime()) > lockStaleAfter {
		os.Remove(path)
		lk, err = os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	}
	if err != nil {
		return nil, fmt.Errorf("json adapter: %s is locked", path)
	}
	return lk, nil
}

// touchLock updates the lock file's timestamp so other adapters know we are
// still alive. Caller must not hold mu.Lock() (flushLoop serialises access).
func (a *Adapter) touchLock() {
	a.lock.WriteString(time.Now().Format(time.RFC3339Nano + "\n"))
}

// flushLoop periodically flushes dirty data and refreshes the lock timestamp.
func (a *Adapter) flushLoop() {
	t := time.NewTicker(flushInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			a.flushIfDirty()
			a.touchLock()
		case <-a.flushStop:
			return
		}
	}
}

// flushIfDirty writes in-memory data to disk if dirty.
func (a *Adapter) flushIfDirty() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.dirty {
		return
	}
	a.write()
	a.dirty = false
}

// write serialises a.data to the file. Caller must hold mu.Lock().
func (a *Adapter) write() error {
	raw, err := json.MarshalIndent(a.data, "", "  ")
	if err != nil {
		return err
	}
	if err := a.file.Truncate(0); err != nil {
		return err
	}
	if _, err := a.file.Seek(0, 0); err != nil {
		return err
	}
	if _, err := a.file.Write(raw); err != nil {
		return err
	}
	return a.file.Sync()
}

// Snapshot returns a copy of the current in-memory snapshot.
func (a *Adapter) Snapshot() (adapter.Snapshot, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := make([]adapter.Role, len(a.data.Role))
	copy(role, a.data.Role)
	user := make([]adapter.User, len(a.data.User))
	copy(user, a.data.User)
	policy := make([]adapter.Policy, len(a.data.Policy))
	copy(policy, a.data.Policy)

	return adapter.Snapshot{Role: role, User: user, Policy: policy}, nil
}

// CreateRole adds a role. Duplicate IDs are silently ignored.
func (a *Adapter) CreateRole(role adapter.Role) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, r := range a.data.Role {
		if r.ID == role.ID {
			return nil
		}
	}
	a.data.Role = append(a.data.Role, role)
	a.dirty = true
	return nil
}

// DeleteRole removes a role by ID. No-op if not found.
func (a *Adapter) DeleteRole(role adapter.Role) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i, r := range a.data.Role {
		if r.ID == role.ID {
			a.data.Role = append(a.data.Role[:i], a.data.Role[i+1:]...)
			a.dirty = true
			return nil
		}
	}
	return nil
}

// LinkUser adds a user-role binding. Duplicate (ID, RoleID) pairs are silently
// ignored — the same user can have multiple roles, each a separate record.
func (a *Adapter) LinkUser(user adapter.User) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, u := range a.data.User {
		if u.ID == user.ID && u.RoleID == user.RoleID {
			return nil
		}
	}
	a.data.User = append(a.data.User, user)
	a.dirty = true
	return nil
}

// DeleteUser removes all role bindings for the given user ID.
func (a *Adapter) DeleteUser(user adapter.User) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	filtered := a.data.User[:0]
	for _, u := range a.data.User {
		if u.ID != user.ID {
			filtered = append(filtered, u)
		}
	}
	a.data.User = filtered
	a.dirty = true
	return nil
}

// UnlinkUser removes a single user-role assignment. No-op if not found.
func (a *Adapter) UnlinkUser(user adapter.User) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i, u := range a.data.User {
		if u.ID == user.ID && u.RoleID == user.RoleID {
			a.data.User = append(a.data.User[:i], a.data.User[i+1:]...)
			a.dirty = true
			return nil
		}
	}
	return nil
}

// CreatePolicy adds a policy. Duplicate IDs are silently ignored.
func (a *Adapter) CreatePolicy(policy adapter.Policy) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, p := range a.data.Policy {
		if p.ID == policy.ID {
			return nil
		}
	}
	a.data.Policy = append(a.data.Policy, policy)
	a.dirty = true
	return nil
}

// DeletePolicy removes a policy by ID. No-op if not found.
func (a *Adapter) DeletePolicy(policyID int64) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i, p := range a.data.Policy {
		if p.ID == policyID {
			a.data.Policy = append(a.data.Policy[:i], a.data.Policy[i+1:]...)
			a.dirty = true
			return nil
		}
	}
	return nil
}

// Close stops the flush goroutine, flushes any remaining dirty data, and
// releases the lock and file handle.
func (a *Adapter) Close() error {
	close(a.flushStop)

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.dirty {
		a.write()
	}
	a.lock.Close()
	os.Remove(a.lockPath)
	return a.file.Close()
}
