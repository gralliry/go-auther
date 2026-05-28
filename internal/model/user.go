package model

import (
	"errors"

	"github.com/gralliry/go-auther/internal/pkg/set"
)

var (
	ErrUserInvalid         = errors.New("user is invalid")
	ErrRoleAlreadyAssigned = errors.New("role already assigned")
	ErrRoleNotAssigned     = errors.New("role not assigned")
)

// User represents a user in the authorization system.
type User struct {
	// immutable field
	id string
	// Valid() verify this field is not nil
	roles *set.CacheSet[*Role]
}

func (u *User) ID() string {
	return u.id
}

func (u *User) Valid() bool {
	return u != nil && u.roles != nil
}

func (u *User) Assign(role *Role) error {
	if !u.Valid() {
		return ErrUserInvalid
	}
	if !role.Valid() {
		return ErrRoleInvalid
	}
	if u.roles.Has(role) {
		return ErrRoleAlreadyAssigned
	}
	u.roles.Add(role)
	return nil
}

func (u *User) Unassign(role *Role) error {
	if !u.Valid() {
		return ErrUserInvalid
	}
	if !role.Valid() {
		return ErrRoleInvalid
	}
	if !u.roles.Has(role) {
		return ErrRoleNotAssigned
	}
	u.roles.Delete(role)
	return nil
}

func (u *User) IsAssign(role *Role) (bool, error) {
	if !u.Valid() {
		return false, ErrUserInvalid
	}
	if !role.Valid() {
		return false, ErrRoleInvalid
	}
	return u.roles.Has(role), nil
}

func (u *User) Enforce(resource Resource) (bool, error) {
	if !u.Valid() {
		return false, ErrUserInvalid
	}
	return u.roles.Any(func(r *Role) bool {
		ok, err := r.Enforce(resource)
		return ok && err == nil
	}), nil
}
