package auther

import "fmt"

// =============================================================================
// Grant API
// =============================================================================

// GrantResource gives a role access to a resource.
// When fromRoleID == toRoleID, the resource is added directly to the role.
// Otherwise, it delegates from an ancestor to a descendant via a grant record.
func (a *Authorizer) GrantResource(fromRoleID, toRoleID, resource string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	fromRole := a.roles[fromRoleID]
	if fromRole == nil {
		return fmt.Errorf("%w: %s", ErrRoleNotFound, fromRoleID)
	}
	toRole := a.roles[toRoleID]
	if toRole == nil {
		return fmt.Errorf("%w: %s", ErrRoleNotFound, toRoleID)
	}

	if !a.isAncestorOrSelf(fromRoleID, toRoleID) {
		return fmt.Errorf("%w: %s is not an ancestor of %s", ErrNotAncestor, fromRoleID, toRoleID)
	}

	// Self-grant: add directly to the role's own resources.
	if fromRoleID == toRoleID {
		if toRole.Resources[resource] {
			return nil
		}
		toRole.Resources[resource] = true
		return a.save()
	}

	for _, g := range fromRole.GrantsOut {
		if g.ToRoleID == toRoleID && g.Resource == resource {
			return fmt.Errorf("%w: %s -> %s %s", ErrDuplicateGrant, fromRoleID, toRoleID, resource)
		}
	}

	grant := RoleGrant{FromRoleID: fromRoleID, ToRoleID: toRoleID, Resource: resource}
	fromRole.GrantsOut = append(fromRole.GrantsOut, grant)
	toRole.GrantsIn = append(toRole.GrantsIn, grant)
	return a.save()
}

// RevokeResource removes a grant and cascades to sub-grants in the subtree.
func (a *Authorizer) RevokeResource(fromRoleID, toRoleID, resource string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.revokeResourceLocked(fromRoleID, toRoleID, resource)
}

func (a *Authorizer) revokeResourceLocked(fromRoleID, toRoleID, resource string) error {
	fromRole := a.roles[fromRoleID]
	toRole := a.roles[toRoleID]
	if fromRole == nil || toRole == nil {
		return fmt.Errorf("%w", ErrGrantNotFound)
	}

	// Self-revoke: remove from the role's own resources.
	if fromRoleID == toRoleID {
		if !toRole.Resources[resource] {
			return ErrGrantNotFound
		}
		delete(toRole.Resources, resource)
		return a.save()
	}

	found := false
	for i, g := range fromRole.GrantsOut {
		if g.ToRoleID == toRoleID && g.Resource == resource {
			fromRole.GrantsOut = append(fromRole.GrantsOut[:i], fromRole.GrantsOut[i+1:]...)
			found = true
			break
		}
	}
	for i, g := range toRole.GrantsIn {
		if g.FromRoleID == fromRoleID && g.Resource == resource {
			toRole.GrantsIn = append(toRole.GrantsIn[:i], toRole.GrantsIn[i+1:]...)
			break
		}
	}
	if !found {
		return fmt.Errorf("%w: %s -> %s %s", ErrGrantNotFound, fromRoleID, toRoleID, resource)
	}

	// Cascade: remove all grants within the subtree for the same resource
	subtree := a.collectSubtree(toRoleID)
	subtreeSet := make(map[string]bool, len(subtree))
	for _, r := range subtree {
		subtreeSet[r.ID] = true
	}
	for _, r := range subtree {
		r.GrantsOut = removeSubtreeGrants(r.GrantsOut, resource, subtreeSet, a.roles)
	}
	return a.save()
}

func removeSubtreeGrants(grants []RoleGrant, resource string, subtreeSet map[string]bool, roles map[string]*RoleNode) []RoleGrant {
	out := grants[:0]
	for _, g := range grants {
		if g.Resource == resource && subtreeSet[g.ToRoleID] {
			if grantee := roles[g.ToRoleID]; grantee != nil {
				grantee.GrantsIn = removeGrantIn(grantee.GrantsIn, g.FromRoleID, resource)
			}
			continue
		}
		out = append(out, g)
	}
	return out
}

func removeGrantIn(grants []RoleGrant, fromRoleID, resource string) []RoleGrant {
	for i, g := range grants {
		if g.FromRoleID == fromRoleID && g.Resource == resource {
			return append(grants[:i], grants[i+1:]...)
		}
	}
	return grants
}

// GetGrantsToRole returns all grants received by the given role.
func (a *Authorizer) GetGrantsToRole(roleID string) ([]RoleGrant, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}
	result := make([]RoleGrant, len(role.GrantsIn))
	copy(result, role.GrantsIn)
	return result, nil
}

// GetGrantsFromRole returns all grants made by the given role.
func (a *Authorizer) GetGrantsFromRole(roleID string) ([]RoleGrant, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}
	result := make([]RoleGrant, len(role.GrantsOut))
	copy(result, role.GrantsOut)
	return result, nil
}

// GetAllGrants returns all grants in the system.
func (a *Authorizer) GetAllGrants() []RoleGrant {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var result []RoleGrant
	seen := make(map[string]bool)
	var walk func(role *RoleNode)
	walk = func(role *RoleNode) {
		for _, g := range role.GrantsOut {
			key := g.FromRoleID + "|" + g.ToRoleID + "|" + g.Resource
			if !seen[key] {
				seen[key] = true
				result = append(result, g)
			}
		}
		for _, child := range role.Children {
			walk(child)
		}
	}
	walk(a.root)
	return result
}
