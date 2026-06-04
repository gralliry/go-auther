package empty

import (
	"github.com/gralliry/go-auther/adapter"
)

// Adapter is a no-op adapter that satisfies the auther.Adapter interface.
// All state is managed in-memory by the Authorizer itself.
type Adapter struct{}

// New creates a new in-memory adapter.
func New() *Adapter { return &Adapter{} }

func (a *Adapter) All() (adapter.Snapshot, error) {
	return adapter.Snapshot{
		Role:   make([]adapter.Role, 0),
		User:   make([]adapter.User, 0),
		Policy: make([]adapter.Policy, 0),
	}, nil
}

func (a *Adapter) CreateRole(role adapter.Role) error { return nil }
func (a *Adapter) DeleteRole(roleID string) error      { return nil }

func (a *Adapter) CreateUser(user adapter.User) error { return nil }
func (a *Adapter) DeleteUser(userID string) error      { return nil }

func (a *Adapter) CreatePolicy(policy adapter.Policy) error { return nil }
func (a *Adapter) DeletePolicy(policyID int64) error        { return nil }
