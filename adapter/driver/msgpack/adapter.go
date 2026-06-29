// Package msgpack provides a msgpack file-backed adapter for Auther.
package msgpack

import (
	"fmt"
	"os"
	"sync"
	"time"

	msgpack "github.com/vmihailenco/msgpack/v5"
	"github.com/gralliry/go-auther/adapter"
)

const (
	flushInterval  = time.Second
	lockStaleAfter = 5 * time.Second
)

// fileData is the on-disk format. Maps enable O(1) dedup; short msgpack tags
// keep the binary compact. The struct is private so the format can evolve
// independently of the public adapter.Snapshot.
type fileData struct {
	Roles    map[string]bool              `msgpack:"r"`
	Users    map[string]map[string]bool   `msgpack:"u"` // uid → rid → true
	Policies map[int64]policyRow          `msgpack:"p"`
}

type policyRow struct {
	Grantor string `msgpack:"o"`
	Grantee string `msgpack:"e"`
	Res     string `msgpack:"r"`
}

// Adapter is a msgpack file-backed store.
type Adapter struct {
	mu     sync.RWMutex
	lock   *os.File
	lockPath string
	dataPath string

	data  fileData
	dirty bool

	flushStop chan struct{}
}

// New opens (or creates) the data file, acquires an exclusive lock, and
// starts the periodic flush.
func New(path string) (*Adapter, error) {
	lockPath := path + ".lock"
	lk, err := acquireLock(lockPath)
	if err != nil {
		return nil, err
	}

	a := &Adapter{
		lock:     lk,
		lockPath: lockPath,
		dataPath: path,
		data: fileData{
			Roles:    make(map[string]bool),
			Users:    make(map[string]map[string]bool),
			Policies: make(map[int64]policyRow),
		},
		flushStop: make(chan struct{}),
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			lk.Close()
			os.Remove(lockPath)
			return nil, fmt.Errorf("msgpack adapter: read: %w", err)
		}
	} else if len(raw) > 0 {
		if err := msgpack.Unmarshal(raw, &a.data); err != nil {
			lk.Close()
			os.Remove(lockPath)
			return nil, fmt.Errorf("msgpack adapter: parse: %w", err)
		}
	}

	go a.flushLoop()
	return a, nil
}

func acquireLock(path string) (*os.File, error) {
	lk, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err == nil {
		return lk, nil
	}
	fi, err := os.Stat(path)
	if err == nil && time.Since(fi.ModTime()) > lockStaleAfter {
		os.Remove(path)
		lk, err = os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	}
	if err != nil {
		return nil, fmt.Errorf("msgpack adapter: %s is locked", path)
	}
	return lk, nil
}

func (a *Adapter) touchLock() {
	a.lock.WriteString(time.Now().Format(time.RFC3339Nano + "\n"))
}

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

func (a *Adapter) flushIfDirty() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.dirty {
		return
	}
	a.write()
	a.dirty = false
}

func (a *Adapter) write() error {
	raw, err := msgpack.Marshal(a.data)
	if err != nil {
		return err
	}
	return os.WriteFile(a.dataPath, raw, 0o644)
}

// Snapshot converts the internal map format to adapter.Snapshot.
func (a *Adapter) Snapshot() (adapter.Snapshot, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	roles := make([]adapter.Role, 0, len(a.data.Roles))
	for id := range a.data.Roles {
		roles = append(roles, adapter.Role{ID: id})
	}

	var users []adapter.User
	for uid, roles := range a.data.Users {
		for rid := range roles {
			users = append(users, adapter.User{ID: uid, RoleID: rid})
		}
	}

	policies := make([]adapter.Policy, 0, len(a.data.Policies))
	for id, p := range a.data.Policies {
		policies = append(policies, adapter.Policy{
			ID: id, GrantorRoleID: p.Grantor, GranteeRoleID: p.Grantee, Resource: p.Res,
		})
	}

	return adapter.Snapshot{Role: roles, User: users, Policy: policies}, nil
}

func (a *Adapter) CreateRole(role adapter.Role) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.data.Roles[role.ID] {
		return nil
	}
	a.data.Roles[role.ID] = true
	a.dirty = true
	return nil
}

func (a *Adapter) DeleteRole(role adapter.Role) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.data.Roles[role.ID] {
		delete(a.data.Roles, role.ID)
		a.dirty = true
	}
	return nil
}

func (a *Adapter) LinkUser(user adapter.User) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	roles := a.data.Users[user.ID]
	if roles != nil && roles[user.RoleID] {
		return nil
	}
	if roles == nil {
		roles = make(map[string]bool)
		a.data.Users[user.ID] = roles
	}
	roles[user.RoleID] = true
	a.dirty = true
	return nil
}

func (a *Adapter) DeleteUser(user adapter.User) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.data.Users[user.ID] != nil {
		delete(a.data.Users, user.ID)
		a.dirty = true
	}
	return nil
}

func (a *Adapter) UnlinkUser(user adapter.User) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	roles := a.data.Users[user.ID]
	if roles != nil && roles[user.RoleID] {
		delete(roles, user.RoleID)
		if len(roles) == 0 {
			delete(a.data.Users, user.ID)
		}
		a.dirty = true
	}
	return nil
}

func (a *Adapter) CreatePolicy(policy adapter.Policy) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, ok := a.data.Policies[policy.ID]; ok {
		return nil
	}
	a.data.Policies[policy.ID] = policyRow{
		Grantor: policy.GrantorRoleID,
		Grantee: policy.GranteeRoleID,
		Res:     policy.Resource,
	}
	a.dirty = true
	return nil
}

func (a *Adapter) DeletePolicy(policyID int64) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, ok := a.data.Policies[policyID]; ok {
		delete(a.data.Policies, policyID)
		a.dirty = true
	}
	return nil
}

func (a *Adapter) Close() error {
	close(a.flushStop)

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.dirty {
		a.write()
	}
	a.lock.Close()
	os.Remove(a.lockPath)
	return nil
}
