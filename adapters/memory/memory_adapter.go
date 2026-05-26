package memory

import (
	"sync"

	"github.com/gralliry/go-auther/snapshot"
)

// Adapter stores policy data in memory. Implements auther.Adapter.
type Adapter struct {
	mu       sync.RWMutex
	snapshot *snapshot.Policy
}

// New creates a new in-memory adapter.
func New() *Adapter {
	return &Adapter{}
}

func (a *Adapter) Load() (*snapshot.Policy, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if a.snapshot == nil {
		return nil, nil
	}
	return a.snapshot.Clone(), nil
}

func (a *Adapter) Save(snapshot *snapshot.Policy) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.snapshot = snapshot.Clone()
	return nil
}

func (a *Adapter) SetRole(role snapshot.Role) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ensure()
	a.snapshot.Roles = append(a.snapshot.Roles, role)
	return nil
}

func (a *Adapter) UnsetRole(role snapshot.Role) error {
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

func (a *Adapter) SetUser(user snapshot.User) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ensure()
	a.snapshot.Users = append(a.snapshot.Users, user)
	return nil
}

func (a *Adapter) UnsetUser(user snapshot.User) error {
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

func (a *Adapter) SetGrant(grant snapshot.Grant) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ensure()
	a.snapshot.Grants = append(a.snapshot.Grants, grant)
	return nil
}

func (a *Adapter) UnsetGrant(grant snapshot.Grant) error {
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

func (a *Adapter) ensure() {
	if a.snapshot == nil {
		a.snapshot = &snapshot.Policy{}
	}
}
