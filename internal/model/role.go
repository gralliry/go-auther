package model

import (
	"errors"

	"github.com/gralliry/go-auther/internal/pkg/set"
)

var (
	ErrRoleInvalid          = errors.New("invalid role")
	ErrRoleAlreadyExist     = errors.New("role already exists")
	ErrRoleNotFound         = errors.New("role not found")
	ErrRoleInsufficient     = errors.New("insufficient role")
	ErrRoleInvalidHierarchy = errors.New("invalid role hierarchy")
	ErrRoleSelfGrant        = errors.New("self grant is not allowed")
)

type Role struct {
	// immutable field
	id string
	// inheritance graph
	srcGrants *set.AutoCacheSet[*Policy]
	tarGrants *set.AutoCacheSet[*Policy]
	// Valid() verify this field is not false
	valid bool
}

func newRole(id string) *Role {
	return &Role{
		id:        id,
		srcGrants: set.NewAutoCacheSet[*Policy](),
		tarGrants: set.NewAutoCacheSet[*Policy](),
		valid:     true,
	}
}

func (r *Role) ID() string {
	return r.id
}

func (r *Role) Valid() bool {
	return r != nil && r.valid
}

func (r *Role) Enforce(resource Resource) (bool, error) {
	if !r.Valid() {
		return false, ErrRoleInvalid
	}
	return r.srcGrants.Any(func(p *Policy) bool {
		return p.Match(resource)
	}), nil
}

func (r *Role) Grant(
	policy *Policy,
	resource Resource,
	grantee *Role,
) (*Policy, error) {
	if !r.Valid() {
		return nil, ErrRoleInvalid
	}
	if !grantee.Valid() {
		return nil, ErrRoleInvalid
	}
	if !r.srcGrants.Has(policy) {
		return nil, ErrPolicyNotFound
	}
	newPolicy, err := policy.delegate(resource)
	if err != nil {
		return nil, err
	}
	r.tarGrants.Add(newPolicy)
	grantee.srcGrants.Add(newPolicy)
	return newPolicy, nil
}

func (r *Role) Revoke(policy *Policy) error {
	if !r.Valid() {
		return ErrRoleInvalid
	}
	if !r.tarGrants.Has(policy) {
		return ErrPolicyNotFound
	}
	r.tarGrants.Delete(policy)
	policy.revoke()
	return nil
}

func (r *Role) Delete() error {
	if !r.Valid() {
		return ErrRoleInvalid
	}
	r.valid = false
	r.srcGrants.Range(func(policy *Policy) {
		// 不要使用 revoke() 方法，否则会递归调用
		policy.valid = false
	})
	r.srcGrants.Clear()
	r.tarGrants.Range(func(policy *Policy) {
		// 使用 revoke() 方法断开与 parent 的关联
		policy.revoke()
	})
	r.tarGrants.Clear()
	return nil
}

func (r *Role) Reviced() []*Policy {
	return r.srcGrants.ToSlice()
}
