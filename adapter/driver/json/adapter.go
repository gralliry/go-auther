// Package json provides a JSON file-backed adapter for Auther.
// All writes are atomic (write to temp file, then rename).
package json

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/gralliry/go-auther/entity"
)

// Adapter is a JSON file-backed entity. All mutations are written atomically
// and the adapter is safe for concurrent use.
type Adapter struct {
	mu   sync.RWMutex
	path string
	data entity.Snapshot
}

// New creates a JSON adapter that persists to the given file path.
// If the file does not exist, it is created with an empty snapshot.
func New(path string) (*Adapter, error) {
	a := &Adapter{path: path}
	if err := a.load(); err != nil {
		return nil, err
	}
	return a, nil
}

// load reads the snapshot from disk. If the file does not exist, it initializes
// an empty snapshot and writes it.
func (a *Adapter) load() error {
	data, err := os.ReadFile(a.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			a.data = entity.Snapshot{
				Role:   make([]entity.Role, 0),
				User:   make([]entity.User, 0),
				Policy: make([]entity.Policy, 0),
			}
			return a.save()
		}
		return err
	}
	return json.Unmarshal(data, &a.data)
}

// save writes the current snapshot atomically to disk using a temp file
// followed by a rename.
func (a *Adapter) save() error {
	if err := os.MkdirAll(filepath.Dir(a.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(a.data, "", "  ")
	if err != nil {
		return fmt.Errorf("json adapter: marshal: %w", err)
	}
	tmp := a.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("json adapter: write: %w", err)
	}
	if err := os.Rename(tmp, a.path); err != nil {
		return fmt.Errorf("json adapter: rename: %w", err)
	}
	return nil
}

// Snapshot returns a copy of the current snapshot.
func (a *Adapter) Snapshot() (entity.Snapshot, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return entity.Snapshot{
		Role:   append([]entity.Role(nil), a.data.Role...),
		User:   append([]entity.User(nil), a.data.User...),
		Policy: append([]entity.Policy(nil), a.data.Policy...),
	}, nil
}

// CreateRole adds a role to the snapshot. Duplicate IDs are silently ignored.
func (a *Adapter) CreateRole(role entity.Role) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, r := range a.data.Role {
		if r.ID == role.ID {
			return nil
		}
	}
	a.data.Role = append(a.data.Role, role)
	return a.save()
}

// DeleteRole removes a role by ID. If the role does not exist, it is a no-op.
func (a *Adapter) DeleteRole(role entity.Role) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i, r := range a.data.Role {
		if r.ID == role.ID {
			a.data.Role = append(a.data.Role[:i], a.data.Role[i+1:]...)
			return a.save()
		}
	}
	return nil
}

// LinkUser adds a user-role binding to the snapshot.
// Duplicate (ID, RoleID) pairs are silently ignored — the same user can have
// multiple roles, each stored as a separate record.
func (a *Adapter) LinkUser(user entity.User) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, u := range a.data.User {
		if u.ID == user.ID && u.RoleID == user.RoleID {
			return nil
		}
	}
	a.data.User = append(a.data.User, user)
	return a.save()
}

// DeleteUser removes all role bindings for the given user ID.
func (a *Adapter) DeleteUser(user entity.User) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	filtered := a.data.User[:0]
	for _, u := range a.data.User {
		if u.ID != user.ID {
			filtered = append(filtered, u)
		}
	}
	a.data.User = filtered
	return a.save()
}

// UnlinkUser removes a single user-role assignment. If the (ID, RoleID) pair
// does not exist, it is a no-op.
func (a *Adapter) UnlinkUser(user entity.User) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i, u := range a.data.User {
		if u.ID == user.ID && u.RoleID == user.RoleID {
			a.data.User = append(a.data.User[:i], a.data.User[i+1:]...)
			return a.save()
		}
	}
	return nil
}

// CreatePolicy adds a policy to the snapshot. Duplicate IDs are silently ignored.
func (a *Adapter) CreatePolicy(policy entity.Policy) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, p := range a.data.Policy {
		if p.ID == policy.ID {
			return nil
		}
	}
	a.data.Policy = append(a.data.Policy, policy)
	return a.save()
}

// DeletePolicy removes a policy by ID. If the policy does not exist, it is a no-op.
func (a *Adapter) DeletePolicy(policyID int64) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i, p := range a.data.Policy {
		if p.ID == policyID {
			a.data.Policy = append(a.data.Policy[:i], a.data.Policy[i+1:]...)
			return a.save()
		}
	}
	return nil
}
