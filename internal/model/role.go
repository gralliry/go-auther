package model

import (
	"errors"

	"github.com/gralliry/go-auther/adapter"
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
	id        string
	srcGrants *set.AutoCacheSet[*Policy]
	tarGrants *set.AutoCacheSet[*Policy]
	valid     bool
	area      *Area
}

func newRole(id string, area *Area) *Role {
	return &Role{
		id:        id,
		srcGrants: set.NewAutoCacheSet[*Policy](),
		tarGrants: set.NewAutoCacheSet[*Policy](),
		valid:     true,
		area:      area,
	}
}

func (r *Role) ID() string  { return r.id }
func (r *Role) Valid() bool { return r != nil && r.valid }

func (r *Role) Enforce(res Resource) (bool, error) {
	r.area.RLock()
	defer r.area.RUnlock()
	return r.enforce(res)
}

func (r *Role) enforce(res Resource) (bool, error) {
	if !r.Valid() {
		return false, ErrRoleInvalid
	}
	return r.srcGrants.Any(func(p *Policy) bool {
		return p.contains(res)
	}), nil
}

func (r *Role) Grant(res Resource, grantee *Role) (*Policy, error) {
	r.area.Lock()
	defer r.area.Unlock()

	if !r.Valid() {
		return nil, ErrRoleInvalid
	}
	if !grantee.Valid() {
		return nil, ErrRoleInvalid
	}
	if r.id == grantee.id {
		return nil, ErrRoleSelfGrant
	}

	parentPolicies := r.srcGrants.Filter(func(p *Policy) bool {
		return p.contains(res)
	})
	if parentPolicies.Length() == 0 {
		return nil, ErrRoleInsufficient
	}

	childPolicies := grantee.srcGrants.Filter(func(p *Policy) bool {
		return p.within(res)
	})

	policy := &Policy{
		id:       r.area.GenerateID(),
		res:      res,
		parents:  parentPolicies,
		children: childPolicies,
		valid:    true,
		area:     r.area,
	}

	parentPolicies.Range(func(parent *Policy) {
		parent.children.Add(policy)
	})
	childPolicies.Range(func(child *Policy) {
		child.parents.Add(policy)
	})

	r.tarGrants.Add(policy)
	grantee.srcGrants.Add(policy)

	r.area.CreatePolicy(adapter.Policy{
		ID:            policy.id,
		Resource:      string(res),
		GrantorRoleID: r.id,
		GranteeRoleID: grantee.id,
	})

	return policy, nil
}

func (r *Role) Revoke(policy *Policy) error {
	r.area.Lock()
	defer r.area.Unlock()

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
	r.area.Lock()
	defer r.area.Unlock()

	if !r.Valid() {
		return ErrRoleInvalid
	}
	r.valid = false

	r.srcGrants.Range(func(p *Policy) { p.revoke() })
	r.tarGrants.Range(func(p *Policy) { p.revoke() })
	r.area.DeleteRole(r.id)
	return nil
}
