package manager

import (
	"github.com/gralliry/go-auther/adapter"
	"github.com/gralliry/go-auther/errors"
	"github.com/gralliry/go-auther/internal/pkg/set"
	"github.com/gralliry/go-auther/internal/resource"
)

// ---------------------------------------------------------------------------
// Role type
// ---------------------------------------------------------------------------

type Role struct {
	srcGrants *set.AutoCacheSet[*Policy]
	tarGrants *set.AutoCacheSet[*Policy]
}

func newRole() *Role {
	return &Role{
		srcGrants: set.NewAutoCacheSet[*Policy](),
		tarGrants: set.NewAutoCacheSet[*Policy](),
	}
}

// enforce checks whether the role has access to the given resource.
// Caller must hold the manager lock.
func (r *Role) enforce(res string) bool {
	return r.srcGrants.Any(func(p *Policy) bool {
		return p.match(res)
	})
}

// grant creates a new policy delegating res from this role to grantee.
// Caller must hold the manager lock.
func (r *Role) grant(res *resource.Resource, grantee *Role, policyID int64) {
	parentPolicies := r.srcGrants.Filter(func(p *Policy) bool {
		return p.contains(res)
	})
	parentNums := parentPolicies.Length()

	childPolicies := grantee.srcGrants.Filter(func(p *Policy) bool {
		return p.within(res)
	})
	subPolicies := grantee.tarGrants.Filter(func(p *Policy) bool {
		return p.within(res)
	})

	policy := &Policy{
		id:       policyID,
		resource: res,
		parents:  parentNums,
		children: childPolicies,
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
}

// ---------------------------------------------------------------------------
// Manager — Role operations
// ---------------------------------------------------------------------------

// CreateRole creates a new role with the given ID and persists it.
func (m *Manager) CreateRole(roleID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.roles[roleID]; ok {
		return errors.ErrRoleExists
	}
	if err := m.adapter.CreateRole(adapter.Role{ID: roleID}); err != nil {
		return err
	}
	role := newRole()
	m.roles[roleID] = role
	return nil
}

// DeleteRole persists the deletion, then revokes all policies and removes from memory.
func (m *Manager) DeleteRole(roleID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	role, ok := m.roles[roleID]
	if !ok {
		return errors.ErrRoleNotFound
	}

	if err := m.adapter.DeleteRole(roleID); err != nil {
		return err
	}

	role.srcGrants.Range(func(p *Policy) {
		p.revoke(func(id int64) { m.adapter.DeletePolicy(id) })
	})
	role.tarGrants.Range(func(p *Policy) {
		p.revoke(func(id int64) { m.adapter.DeletePolicy(id) })
	})

	delete(m.roles, roleID)
	return nil
}

// Grant delegates a resource from grantorRole to granteeRole.
func (m *Manager) Grant(grantorID, res, granteeID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	grantor, ok := m.roles[grantorID]
	if !ok {
		return errors.ErrRoleNotFound
	}
	grantee, ok := m.roles[granteeID]
	if !ok {
		return errors.ErrRoleNotFound
	}
	if grantorID == granteeID {
		return errors.ErrRoleSelfGrant
	}

	r := resource.NewResource(res)

	if !grantor.srcGrants.Any(func(p *Policy) bool {
		return p.contains(r)
	}) {
		return errors.ErrRoleInsufficient
	}

	policyID := m.generateID()

	if err := m.adapter.CreatePolicy(adapter.Policy{
		ID:            policyID,
		Resource:      r.String(),
		GrantorRoleID: grantorID,
		GranteeRoleID: granteeID,
	}); err != nil {
		return err
	}

	grantor.grant(r, grantee, policyID)
	return nil
}

// Revoke removes a policy previously granted by the specified role, identified
// by its resource pattern. The revocation cascades to orphaned children.
func (m *Manager) Revoke(roleID, res string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	role, ok := m.roles[roleID]
	if !ok {
		return errors.ErrRoleNotFound
	}

	// Look up the policy by resource string.
	r := resource.NewResource(res)
	target := r.String()
	var found *Policy
	role.tarGrants.Range(func(p *Policy) {
		if p.resource.String() == target {
			found = p
		}
	})
	if found == nil {
		return errors.ErrPolicyNotFound
	}

	if err := m.adapter.DeletePolicy(found.id); err != nil {
		return err
	}
	skipped := found.id

	role.tarGrants.Delete(found)
	found.revoke(func(id int64) {
		if id != skipped {
			m.adapter.DeletePolicy(id)
		}
	})
	return nil
}

// EnforceByRole checks whether a role has access to a resource.
func (m *Manager) EnforceByRole(roleID, res string) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	role, ok := m.roles[roleID]
	if !ok {
		return false, errors.ErrRoleNotFound
	}
	return role.enforce(res), nil
}
