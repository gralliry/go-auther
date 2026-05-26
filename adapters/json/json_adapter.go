// Package json provides a JSON file-backed adapter for policy persistence.
//
// Usage:
//
//	adapter := json.New("policy.json")
//	a, _ := auther.NewAuthorizer(adapter)
package json

import (
	"os"
	"sync"

	"github.com/goccy/go-json"

	"github.com/gralliry/go-auther/snapshot"
)

// Adapter is a JSON file-backed adapter for Auther policy persistence.
// Writes are atomic via temp file + rename.
//
// Incremental methods maintain an in-memory snapshot and trigger a full
// file write on each call.
type Adapter struct {
	filePath string
	mu       sync.Mutex
	snap     *snapshot.Policy // cached for incremental modifications
}

// New creates a new JSON adapter that persists to the given path.
func New(filePath string) *Adapter {
	return &Adapter{filePath: filePath}
}

// Load reads the policy snapshot from the JSON file.
// Returns nil if the file does not exist.
func (ja *Adapter) Load() (*snapshot.Policy, error) {
	ja.mu.Lock()
	defer ja.mu.Unlock()

	data, err := os.ReadFile(ja.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var snap snapshot.Policy
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	ja.snap = &snap
	return snap.Clone(), nil
}

// Save persists the policy snapshot and updates the cache.
// Uses atomic write: writes to temp file, then renames.
func (ja *Adapter) Save(snapshot *snapshot.Policy) error {
	ja.mu.Lock()
	defer ja.mu.Unlock()

	ja.snap = snapshot.Clone()
	return ja.writeLocked()
}

func (ja *Adapter) writeLocked() error {
	data, err := json.MarshalIndent(ja.snap, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := ja.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, ja.filePath)
}

// Incremental methods modify the cached snapshot and write the full file.
// The Authorizer calls these after completing the in-memory mutation.

func (ja *Adapter) SetRole(role snapshot.Role) error {
	ja.mu.Lock()
	defer ja.mu.Unlock()
	if ja.snap == nil {
		ja.snap = &snapshot.Policy{}
	}
	ja.snap.Roles = append(ja.snap.Roles, role)
	return ja.writeLocked()
}

func (ja *Adapter) UnsetRole(role snapshot.Role) error {
	ja.mu.Lock()
	defer ja.mu.Unlock()
	if ja.snap == nil {
		return nil
	}
	for i, r := range ja.snap.Roles {
		if r.ID == role.ID {
			ja.snap.Roles = append(ja.snap.Roles[:i], ja.snap.Roles[i+1:]...)
			break
		}
	}
	return ja.writeLocked()
}

func (ja *Adapter) SetUser(user snapshot.User) error {
	ja.mu.Lock()
	defer ja.mu.Unlock()
	if ja.snap == nil {
		ja.snap = &snapshot.Policy{}
	}
	ja.snap.Users = append(ja.snap.Users, user)
	return ja.writeLocked()
}

func (ja *Adapter) UnsetUser(user snapshot.User) error {
	ja.mu.Lock()
	defer ja.mu.Unlock()
	if ja.snap == nil {
		return nil
	}
	for i, u := range ja.snap.Users {
		if u.ID == user.ID {
			ja.snap.Users = append(ja.snap.Users[:i], ja.snap.Users[i+1:]...)
			break
		}
	}
	return ja.writeLocked()
}

func (ja *Adapter) SetGrant(grant snapshot.Grant) error {
	ja.mu.Lock()
	defer ja.mu.Unlock()
	if ja.snap == nil {
		ja.snap = &snapshot.Policy{}
	}
	ja.snap.Grants = append(ja.snap.Grants, grant)
	return ja.writeLocked()
}

func (ja *Adapter) UnsetGrant(grant snapshot.Grant) error {
	ja.mu.Lock()
	defer ja.mu.Unlock()
	if ja.snap == nil {
		return nil
	}
	for i, g := range ja.snap.Grants {
		if g.FromRoleID == grant.FromRoleID && g.ToRoleID == grant.ToRoleID && g.Resource == grant.Resource {
			ja.snap.Grants = append(ja.snap.Grants[:i], ja.snap.Grants[i+1:]...)
			break
		}
	}
	return ja.writeLocked()
}
