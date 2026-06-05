package model

import (
	"github.com/gralliry/go-auther/adapter"
	"github.com/gralliry/go-auther/errors"
	"github.com/gralliry/go-auther/internal/pkg/set"
)

type User struct {
	id    string
	roles *set.AutoCacheSet[*Role]
	valid bool
	area  *Area
}

func newUser(id string, area *Area) *User {
	return &User{
		id:    id,
		roles: set.NewAutoCacheSet[*Role](),
		valid: true,
		area:  area,
	}
}

// ID returns the user's unique identifier.
func (u *User) ID() string { return u.id }

// Valid reports whether the user is still active.
func (u *User) Valid() bool { return u != nil && u.valid }

// Assign adds a role to the user. The role must be valid and not already assigned.
func (u *User) Assign(role *Role) error {
	u.area.Lock()
	defer u.area.Unlock()

	if !u.Valid() {
		return errors.ErrUserInvalid
	}
	if !role.Valid() {
		return errors.ErrRoleInvalid
	}
	if u.roles.Has(role) {
		return errors.ErrRoleAlreadyAssigned
	}
	u.roles.Add(role)
	u.area.CreateUser(adapter.User{ID: u.id, RoleID: role.id})
	return nil
}

// Unassign removes a role from the user. The role must be currently assigned.
func (u *User) Unassign(role *Role) error {
	u.area.Lock()
	defer u.area.Unlock()

	if !u.Valid() {
		return errors.ErrUserInvalid
	}
	if !role.Valid() {
		return errors.ErrRoleInvalid
	}
	if !u.roles.Has(role) {
		return errors.ErrRoleNotAssigned
	}
	u.roles.Delete(role)
	return nil
}

// IsAssign reports whether the user currently has the given role.
func (u *User) IsAssign(role *Role) (bool, error) {
	u.area.RLock()
	defer u.area.RUnlock()

	if !u.Valid() {
		return false, errors.ErrUserInvalid
	}
	if !role.Valid() {
		return false, errors.ErrRoleInvalid
	}
	return u.roles.Has(role), nil
}

// Enforce checks whether the user has access to the given resource through any assigned role.
func (u *User) Enforce(resource string) (bool, error) {
	u.area.RLock()
	defer u.area.RUnlock()

	if !u.Valid() {
		return false, errors.ErrUserInvalid
	}
	return u.roles.Any(func(r *Role) bool {
		ok, err := r.enforce(resource)
		return ok && err == nil
	}), nil
}

// Delete marks the user as invalid and removes all role assignments.
func (u *User) Delete() error {
	u.area.Lock()
	defer u.area.Unlock()

	if !u.Valid() {
		return errors.ErrUserInvalid
	}
	u.valid = false
	u.roles.Clear()
	u.area.DeleteUser(u.id)
	return nil
}
