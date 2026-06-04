package model

import (
	"errors"

	"github.com/gralliry/go-auther/adapter"
	"github.com/gralliry/go-auther/internal/pkg/set"
)

var (
	ErrUserInvalid         = errors.New("user is invalid")
	ErrRoleAlreadyAssigned = errors.New("role already assigned")
	ErrRoleNotAssigned     = errors.New("role not assigned")
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

func (u *User) ID() string  { return u.id }
func (u *User) Valid() bool { return u != nil && u.valid }

func (u *User) Assign(role *Role) error {
	u.area.Lock()
	defer u.area.Unlock()

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
	u.area.CreateUser(adapter.User{ID: u.id, RoleID: role.id})
	return nil
}

func (u *User) Unassign(role *Role) error {
	u.area.Lock()
	defer u.area.Unlock()

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
	u.area.RLock()
	defer u.area.RUnlock()

	if !u.Valid() {
		return false, ErrUserInvalid
	}
	if !role.Valid() {
		return false, ErrRoleInvalid
	}
	return u.roles.Has(role), nil
}

func (u *User) Enforce(res Resource) (bool, error) {
	u.area.RLock()
	defer u.area.RUnlock()

	if !u.Valid() {
		return false, ErrUserInvalid
	}
	return u.roles.Any(func(r *Role) bool {
		ok, err := r.enforce(res)
		return ok && err == nil
	}), nil
}

func (u *User) Delete() error {
	u.area.Lock()
	defer u.area.Unlock()

	if !u.Valid() {
		return ErrUserInvalid
	}
	u.valid = false
	u.roles.Clear()
	u.area.DeleteUser(u.id)
	return nil
}
