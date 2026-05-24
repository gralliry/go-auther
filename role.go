package auther

import "fmt"

// =============================================================================
// Role API
// =============================================================================

// CreateRole creates a new sub-role under the given parent role.
func (a *Authorizer) CreateRole(parentID, roleID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.roles[roleID]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicateRole, roleID)
	}

	parent := a.roles[parentID]
	if parent == nil {
		return fmt.Errorf("%w: %s", ErrRoleNotFound, parentID)
	}

	role := &RoleNode{
		ID:        roleID,
		Parent:    parent,
		Children:  make(map[string]*RoleNode),
		Resources: make(map[string]bool),
		Users:     make(map[string]*UserNode),
	}
	a.roles[roleID] = role
	parent.Children[roleID] = role

	return a.save()
}

// DeleteRole deletes a role and cascades to all sub-roles and their users.
// Grants involving deleted roles are cleaned up from surviving roles.
// The root role cannot be deleted.
func (a *Authorizer) DeleteRole(roleID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if roleID == "root" {
		return ErrRootRoleDelete
	}

	target := a.roles[roleID]
	if target == nil {
		return fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}

	subtree := a.collectSubtree(roleID)
	subtreeSet := make(map[string]bool, len(subtree))
	for _, r := range subtree {
		subtreeSet[r.ID] = true
	}

	for _, r := range subtree {
		for userID := range r.Users {
			delete(a.users, userID)
		}
	}

	for _, r := range a.roles {
		if subtreeSet[r.ID] {
			continue
		}
		r.GrantsIn = filterGrantsByFrom(r.GrantsIn, subtreeSet)
		r.GrantsOut = filterGrantsByTo(r.GrantsOut, subtreeSet)
	}

	if target.Parent != nil {
		delete(target.Parent.Children, roleID)
	}
	for _, r := range subtree {
		delete(a.roles, r.ID)
	}

	return a.save()
}

// GetRole returns information about a role.
func (a *Authorizer) GetRole(roleID string) (*RoleInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}
	return roleToInfo(role), nil
}

// GetAllRoles returns all roles in the system.
func (a *Authorizer) GetAllRoles() []*RoleInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var result []*RoleInfo
	var walk func(role *RoleNode)
	walk = func(role *RoleNode) {
		result = append(result, roleToInfo(role))
		for _, child := range role.Children {
			walk(child)
		}
	}
	walk(a.root)
	return result
}

// GetEffectiveRoleResources returns all resource patterns effective for a role.
// Includes the role's own resources and explicit GrantsIn. No auto-inheritance.
func (a *Authorizer) GetEffectiveRoleResources(roleID string) ([]string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}

	seen := make(map[string]bool)
	var result []string
	for res := range role.Resources {
		if !seen[res] {
			seen[res] = true
			result = append(result, res)
		}
	}
	for _, g := range role.GrantsIn {
		if !seen[g.Resource] {
			seen[g.Resource] = true
			result = append(result, g.Resource)
		}
	}
	return result, nil
}

// addResourceToRole is the internal, unchecked direct resource assignment.
func (a *Authorizer) addResourceToRole(roleID, resource string) error {
	role := a.roles[roleID]
	if role == nil {
		return fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}
	role.Resources[resource] = true
	return nil
}

// removeResourceFromRole is the internal, unchecked direct resource removal.
func (a *Authorizer) removeResourceFromRole(roleID, resource string) error {
	role := a.roles[roleID]
	if role == nil {
		return fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}
	delete(role.Resources, resource)
	return nil
}

// =============================================================================
// Helpers
// =============================================================================

func roleToInfo(role *RoleNode) *RoleInfo {
	info := &RoleInfo{
		ID:         role.ID,
		ParentID:   "",
		Resources:  make([]string, 0, len(role.Resources)),
		SubRoleIDs: make([]string, 0, len(role.Children)),
		UserIDs:    make([]string, 0, len(role.Users)),
		GrantsIn:   append([]RoleGrant(nil), role.GrantsIn...),
		GrantsOut:  append([]RoleGrant(nil), role.GrantsOut...),
	}
	if role.Parent != nil {
		info.ParentID = role.Parent.ID
	}
	for res := range role.Resources {
		info.Resources = append(info.Resources, res)
	}
	for childID := range role.Children {
		info.SubRoleIDs = append(info.SubRoleIDs, childID)
	}
	for userID := range role.Users {
		info.UserIDs = append(info.UserIDs, userID)
	}
	return info
}

func filterGrantsByFrom(grants []RoleGrant, excluded map[string]bool) []RoleGrant {
	out := grants[:0]
	for _, g := range grants {
		if excluded[g.FromRoleID] {
			continue
		}
		out = append(out, g)
	}
	return out
}

func filterGrantsByTo(grants []RoleGrant, excluded map[string]bool) []RoleGrant {
	out := grants[:0]
	for _, g := range grants {
		if excluded[g.ToRoleID] {
			continue
		}
		out = append(out, g)
	}
	return out
}
