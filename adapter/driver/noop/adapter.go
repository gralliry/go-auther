package noop

import (
	"github.com/gralliry/go-auther/entity"
)

// Adapter satisfies the auther.Adapter interface with no-op methods.
// All state is managed in-memory by the Manager.
type Adapter struct{}

// New creates a new in-memory adapter.
func New() *Adapter { return &Adapter{} }

// Snapshot returns an empty snapshot. State is only kept in-memory by the Manager.
func (a *Adapter) Snapshot() (entity.Snapshot, error) {
	return entity.Snapshot{
		Role:   make([]entity.Role, 0),
		User:   make([]entity.User, 0),
		Policy: make([]entity.Policy, 0),
	}, nil
}

// CreateRole is a no-op — state lives in the Manager.
func (a *Adapter) CreateRole(role entity.Role) error { return nil }
// DeleteRole is a no-op.
func (a *Adapter) DeleteRole(role entity.Role) error      { return nil }

// LinkUser is a no-op.
func (a *Adapter) LinkUser(user entity.User) error { return nil }
// RemoveUser is a no-op.
func (a *Adapter) RemoveUser(user entity.User) error      { return nil }
// UnlinkUser is a no-op.
func (a *Adapter) UnlinkUser(user entity.User) error     { return nil }

// CreatePolicy is a no-op.
func (a *Adapter) CreatePolicy(policy entity.Policy) error { return nil }
// DeletePolicy is a no-op.
func (a *Adapter) DeletePolicy(policyID int64) error        { return nil }
