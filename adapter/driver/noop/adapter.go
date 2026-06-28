package noop

import (
	"github.com/gralliry/go-auther/adapter"
)

// Adapter satisfies the auther.Adapter interface with no-op methods.
// All state is managed in-memory by the Manager.
type Adapter struct{}

// New creates a new in-memory adapter.
func New() *Adapter { return &Adapter{} }

// All returns an empty snapshot. State is only kept in-memory by the Manager.
func (a *Adapter) All() (adapter.Snapshot, error) {
	return adapter.Snapshot{
		Role:   make([]adapter.Role, 0),
		User:   make([]adapter.User, 0),
		Policy: make([]adapter.Policy, 0),
	}, nil
}

// CreateRole is a no-op — state lives in the Manager.
func (a *Adapter) CreateRole(role adapter.Role) error { return nil }
// DeleteRole is a no-op.
func (a *Adapter) DeleteRole(role adapter.Role) error      { return nil }

// CreateUser is a no-op.
func (a *Adapter) CreateUser(user adapter.User) error { return nil }
// DeleteUser is a no-op.
func (a *Adapter) DeleteUser(user adapter.User) error      { return nil }
// UnassignUser is a no-op.
func (a *Adapter) UnassignUser(user adapter.User) error     { return nil }

// CreatePolicy is a no-op.
func (a *Adapter) CreatePolicy(policy adapter.Policy) error { return nil }
// DeletePolicy is a no-op.
func (a *Adapter) DeletePolicy(policyID int64) error        { return nil }
