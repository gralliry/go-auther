package model

import (
	"github.com/gralliry/go-auther/adapter"
	"github.com/gralliry/go-auther/errors"
	"github.com/gralliry/go-auther/internal/pkg/set"
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

// ID returns the role's unique identifier.
func (r *Role) ID() string { return r.id }

// Valid reports whether the role is still active.
func (r *Role) Valid() bool { return r != nil && r.valid }

// Enforce checks whether the role has access to the given resource.
func (r *Role) Enforce(res string) (bool, error) {
	r.area.RLock()
	defer r.area.RUnlock()
	return r.enforce(res)
}

func (r *Role) enforce(res string) (bool, error) {
	if !r.Valid() {
		return false, errors.ErrRoleInvalid
	}
	return r.srcGrants.Any(func(p *Policy) bool {
		return p.match(res)
	}), nil
}

// Grant delegates a resource from this role to the grantee. The grantor must already
// hold a policy that contains the resource. Self-grant is rejected. Returns the newly
// created policy.
func (r *Role) Grant(res *Resource, grantee *Role) (*Policy, error) {
	r.area.Lock()
	defer r.area.Unlock()

	if !r.Valid() {
		return nil, errors.ErrRoleInvalid
	}
	if !grantee.Valid() {
		return nil, errors.ErrGranteeInvalid
	}
	if r.id == grantee.id {
		return nil, errors.ErrRoleSelfGrant
	}

	parentPolicies := r.srcGrants.Filter(func(p *Policy) bool {
		return p.contains(res)
	})
	parentNums := parentPolicies.Length()
	if parentNums == 0 {
		return nil, errors.ErrRoleInsufficient
	}

	childPolicies := grantee.srcGrants.Filter(func(p *Policy) bool {
		return p.within(res)
	})
	subPolicies := grantee.tarGrants.Filter(func(p *Policy) bool {
		return p.within(res)
	})

	policy := &Policy{
		id:       r.area.GenerateID(),
		resource: res,
		parents:  parentNums,
		children: childPolicies,
		area:     r.area,
	}

	parentPolicies.Range(func(parent *Policy) {
		parent.children.Add(policy)
	})
	childPolicies.Range(func(child *Policy) {
		child.parents++
	})
	subPolicies.Range(func(sub *Policy) {
		policy.children.Add(sub)
		sub.parents++
	})

	r.tarGrants.Add(policy)
	grantee.srcGrants.Add(policy)

	r.area.CreatePolicy(adapter.Policy{
		ID:            policy.id,
		Resource:      res.String(),
		GrantorRoleID: r.id,
		GranteeRoleID: grantee.id,
	})

	return policy, nil
}

// Revoke revokes a policy previously granted by this role. The policy is removed
// from the role's outgoing grants and cascaded to orphaned children.
func (r *Role) Revoke(policy *Policy) error {
	r.area.Lock()
	defer r.area.Unlock()

	if !r.Valid() {
		return errors.ErrRoleInvalid
	}
	if !r.tarGrants.Has(policy) {
		return errors.ErrPolicyNotFound
	}
	r.tarGrants.Delete(policy)
	policy.revoke()
	return nil
}

// Delete marks the role as invalid and revokes all its incoming and outgoing policies.
func (r *Role) Delete() error {
	r.area.Lock()
	defer r.area.Unlock()

	if !r.Valid() {
		return errors.ErrRoleInvalid
	}
	r.valid = false

	r.srcGrants.Range(func(p *Policy) { p.revoke() })
	r.tarGrants.Range(func(p *Policy) { p.revoke() })
	r.area.DeleteRole(r.id)
	return nil
}
