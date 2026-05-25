package memoryadapter

import (
	"sync"

	"auther/snapshot"
)

// MemoryAdapter stores policy data in memory. Implements auther.Adapter.
type MemoryAdapter struct {
	mu       sync.RWMutex
	snapshot *snapshot.Policy
}

// NewMemoryAdapter creates a new in-memory adapter.
func NewMemoryAdapter() *MemoryAdapter {
	return &MemoryAdapter{}
}

func (a *MemoryAdapter) Load() (*snapshot.Policy, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.snapshot == nil {
		return nil, nil
	}
	return copySnapshot(a.snapshot), nil
}

func (a *MemoryAdapter) Save(snapshot *snapshot.Policy) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.snapshot = copySnapshot(snapshot)
	return nil
}

func (a *MemoryAdapter) SetRole(role snapshot.Role) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ensure()
	a.snapshot.Roles = append(a.snapshot.Roles, role)
	return nil
}

func (a *MemoryAdapter) UnsetRole(role snapshot.Role) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.snapshot == nil {
		return nil
	}
	for i, r := range a.snapshot.Roles {
		if r.ID == role.ID {
			a.snapshot.Roles = append(a.snapshot.Roles[:i], a.snapshot.Roles[i+1:]...)
			break
		}
	}
	return nil
}

func (a *MemoryAdapter) SetUser(user snapshot.User) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ensure()
	a.snapshot.Users = append(a.snapshot.Users, user)
	return nil
}

func (a *MemoryAdapter) UnsetUser(user snapshot.User) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.snapshot == nil {
		return nil
	}
	for i, u := range a.snapshot.Users {
		if u.ID == user.ID {
			a.snapshot.Users = append(a.snapshot.Users[:i], a.snapshot.Users[i+1:]...)
			break
		}
	}
	return nil
}

func (a *MemoryAdapter) SetGrant(grant snapshot.Grant) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ensure()
	a.snapshot.Grants = append(a.snapshot.Grants, grant)
	return nil
}

func (a *MemoryAdapter) UnsetGrant(grant snapshot.Grant) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.snapshot == nil {
		return nil
	}
	for i, g := range a.snapshot.Grants {
		if g.FromRoleID == grant.FromRoleID && g.ToRoleID == grant.ToRoleID && g.Resource == grant.Resource {
			a.snapshot.Grants = append(a.snapshot.Grants[:i], a.snapshot.Grants[i+1:]...)
			break
		}
	}
	return nil
}

func (a *MemoryAdapter) ensure() {
	if a.snapshot == nil {
		a.snapshot = &snapshot.Policy{}
	}
}

func copySnapshot(s *snapshot.Policy) *snapshot.Policy {
	if s == nil {
		return nil
	}
	c := &snapshot.Policy{}
	c.Roles = make([]snapshot.Role, len(s.Roles))
	copy(c.Roles, s.Roles)
	c.Users = make([]snapshot.User, len(s.Users))
	copy(c.Users, s.Users)
	c.Grants = make([]snapshot.Grant, len(s.Grants))
	copy(c.Grants, s.Grants)
	return c
}
