package auther

import (
	"fmt"

	"github.com/gralliry/go-auther/internal/model"
	"github.com/gralliry/go-auther/snapshot"
)

// CreateRole creates a new child role under the specified parent role.
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

	role := &model.RoleNode{
		ID:         roleID,
		Parent:     parent,
		Children:   make(map[string]*model.RoleNode),
		GrantedMap: make(map[string]bool),
		Users:      make(map[string]*model.UserNode),
	}
	a.roles[roleID] = role
	parent.Children[roleID] = role

	return a.adapter.SetRole(snapshot.Role{ID: roleID, ParentID: parentID})
}

// DeleteRole deletes a role, cascading to all sub-roles and their users.
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

	// Collect the entire subtree to delete.
	subtree := target.Subtree()
	excluded := make(map[string]bool, len(subtree))
	for _, r := range subtree {
		excluded[r.ID] = true
	}

	// Remove all users in the subtree.
	for _, r := range subtree {
		for userID := range r.Users {
			delete(a.users, userID)
		}
	}

	// Clean up grants and rebuild GrantedMap for surviving roles.
	for _, r := range a.roles {
		if excluded[r.ID] {
			continue
		}
		r.GrantsIn = model.FilterByFrom(r.GrantsIn, excluded)
		r.GrantsOut = model.FilterByTo(r.GrantsOut, excluded)
		r.GrantedMap = make(map[string]bool)
		for _, g := range r.GrantsIn {
			r.GrantedMap[string(g.Resource)] = true
		}
		r.ResetMatchCache()
	}

	// Detach from parent and remove all roles in the subtree.
	if target.Parent != nil {
		delete(target.Parent.Children, roleID)
	}
	for _, r := range subtree {
		delete(a.roles, r.ID)
	}

	return a.save()
}

// GetRole returns detailed information for the specified role.
func (a *Authorizer) GetRole(roleID string) (*model.RoleInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}
	return role.ToInfo(), nil
}

// GetAllRoles returns information for all roles in the system, in BFS order.
func (a *Authorizer) GetAllRoles() []*model.RoleInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var result []*model.RoleInfo
	root := a.roles["root"]
	if root == nil {
		return nil
	}
	queue := []*model.RoleNode{root}
	for len(queue) > 0 {
		role := queue[0]
		queue = queue[1:]
		result = append(result, role.ToInfo())
		for _, child := range role.Children {
			queue = append(queue, child)
		}
	}
	return result
}

// GetSubRoles returns the direct child roles of the specified role.
func (a *Authorizer) GetSubRoles(roleID string) ([]*model.RoleInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}

	result := make([]*model.RoleInfo, 0, len(role.Children))
	for _, child := range role.Children {
		result = append(result, child.ToInfo())
	}
	return result, nil
}

// GetResource returns the effective resource patterns currently granted to the role.
func (a *Authorizer) GetResource(roleID string) ([]string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}

	return role.Resources(), nil
}
