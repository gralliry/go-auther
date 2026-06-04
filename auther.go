package auther

import (
	"github.com/gralliry/go-auther/adapter"
	"github.com/gralliry/go-auther/internal/model"
)

type (
	Resource = model.Resource
	Role     = model.Role
	User     = model.User
	Policy   = model.Policy
	Manager  = model.Manager
	Adapter  = adapter.Adapter
)

// NewResource creates a validated and normalized resource from a raw path.
func NewResource(raw string) Resource {
	return model.NewResource(raw)
}

// Sentinel errors.
var (
	ErrUserInvalid            = model.ErrUserInvalid
	ErrRoleInvalid            = model.ErrRoleInvalid
	ErrRoleAlreadyAssigned    = model.ErrRoleAlreadyAssigned
	ErrRoleNotAssigned        = model.ErrRoleNotAssigned
	ErrRoleSelfGrant          = model.ErrRoleSelfGrant
	ErrRoleInsufficient       = model.ErrRoleInsufficient
	ErrPolicyNotFound         = model.ErrPolicyNotFound
)

// New creates a new Manager with the given adapter.
func New(adapter Adapter) (*Manager, error) {
	return model.New(adapter)
}
