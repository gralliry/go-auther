package auther

import (
	"fmt"

	"github.com/gralliry/go-auther/internal/model"
	"github.com/gralliry/go-auther/internal/resource"
	"github.com/gralliry/go-auther/snapshot"
)

// Grant grants a resource pattern from an ancestor role to a descendant role.
//
// The grantor must be an ancestor of the grantee. Self-grant is not allowed.
func (a *Authorizer) Grant(fromRoleID, toRoleID, resource string) error {
	res, err := normalizeRes(resource)
	if err != nil {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if fromRoleID == toRoleID {
		return fmt.Errorf("%w: self-grant is not allowed", ErrNotAncestor)
	}
	fromRole := a.roles[fromRoleID]
	if fromRole == nil {
		return fmt.Errorf("%w: %s", ErrRoleNotFound, fromRoleID)
	}
	toRole := a.roles[toRoleID]
	if toRole == nil {
		return fmt.Errorf("%w: %s", ErrRoleNotFound, toRoleID)
	}
	if !toRole.HasAncestor(fromRoleID) {
		return fmt.Errorf("%w: %s is not an ancestor of %s", ErrNotAncestor, fromRoleID, toRoleID)
	}

	for _, g := range fromRole.GrantsOut {
		if g.ToRoleID == toRoleID && g.Resource == res {
			return fmt.Errorf("%w: %s -> %s %s", ErrDuplicateGrant, fromRoleID, toRoleID, res)
		}
	}

	grant := &model.GrantNode{FromRoleID: fromRoleID, ToRoleID: toRoleID, Resource: res}
	fromRole.GrantsOut = append(fromRole.GrantsOut, grant)
	toRole.GrantsIn = append(toRole.GrantsIn, grant)
	toRole.GrantedMap[string(res)] = true
	toRole.ResetMatchCache()
	return a.adapter.SetGrant(snapshot.Grant{FromRoleID: fromRoleID, ToRoleID: toRoleID, Resource: string(res)})
}

// Revoke revokes a grant and cascades cleanup to invalidated sub-grants.
func (a *Authorizer) Revoke(fromRoleID, toRoleID, resource string) error {
	res, err := normalizeRes(resource)
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	if fromRoleID == toRoleID {
		return fmt.Errorf("%w: self-revoke is not allowed", ErrGrantNotFound)
	}
	fromRole := a.roles[fromRoleID]
	toRole := a.roles[toRoleID]
	if fromRole == nil || toRole == nil {
		return fmt.Errorf("%w", ErrGrantNotFound)
	}
	return a.revokeDelegatedLocked(fromRole, toRole, res)
}

// revokeDelegatedLocked removes a grant and cascades cleanup to descendant roles
// whose sub-delegations are no longer covered.
func (a *Authorizer) revokeDelegatedLocked(fromRole, toRole *model.RoleNode, resource resource.Resource) error {
	found := false
	for i, g := range fromRole.GrantsOut {
		if g.ToRoleID == toRole.ID && g.Resource == resource {
			fromRole.GrantsOut = append(fromRole.GrantsOut[:i], fromRole.GrantsOut[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("%w: %s -> %s %s", ErrGrantNotFound, fromRole.ID, toRole.ID, resource)
	}

	for i, g := range toRole.GrantsIn {
		if g.FromRoleID == fromRole.ID && g.Resource == resource {
			toRole.GrantsIn = append(toRole.GrantsIn[:i], toRole.GrantsIn[i+1:]...)
			break
		}
	}
	if !model.HasGrant(toRole.GrantsIn, resource) {
		delete(toRole.GrantedMap, string(resource))
	}
	toRole.ResetMatchCache()

	// Cascade: for each descendant, remove sub-grants whose source is no longer covered.
	for _, r := range toRole.Subtree() {
		r.GrantsOut = model.RemoveGrantsAndCleanup(r.GrantsOut, a.roles)
		r.ResetMatchCache()
	}
	return a.save()
}

// GetGrantsTo returns all grants received by the specified role.
func (a *Authorizer) GetGrantsTo(roleID string) ([]*model.GrantNode, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}
	return append([]*model.GrantNode(nil), role.GrantsIn...), nil
}

// GetGrantsFrom returns all grants issued by the specified role.
func (a *Authorizer) GetGrantsFrom(roleID string) ([]*model.GrantNode, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}
	return append([]*model.GrantNode(nil), role.GrantsOut...), nil
}
