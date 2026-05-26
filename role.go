package auther

import (
	"fmt"

	"github.com/gralliry/go-auther/internal/model"
	"github.com/gralliry/go-auther/snapshot"
)

// CreateRole 在指定父角色下创建一个新的子角色。
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

// DeleteRole 删除指定角色，级联删除其所有子角色及关联用户。
// 涉及已删除角色的授权记录会从幸存角色中清理。根角色不可删除。
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

	// 收集待删除的子树。
	subtree := target.Subtree()
	excluded := make(map[string]bool, len(subtree))
	for _, r := range subtree {
		excluded[r.ID] = true
	}

	// 移除子树中所有用户。
	for _, r := range subtree {
		for userID := range r.Users {
			delete(a.users, userID)
		}
	}

	// 清理幸存角色中与已删除角色相关的授权记录并重建 GrantedMap。
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

	// 解除父角色引用后删除子树中的所有角色。
	if target.Parent != nil {
		delete(target.Parent.Children, roleID)
	}
	for _, r := range subtree {
		delete(a.roles, r.ID)
	}

	return a.save()
}

// GetRole 返回指定角色的详细信息。
func (a *Authorizer) GetRole(roleID string) (*model.RoleInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}
	return role.ToInfo(), nil
}

// GetAllRoles 返回系统中所有角色的信息列表。
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

// GetSubRoles 返回指定角色的直接子角色列表。
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

// GetResource 返回角色当前生效的所有资源权限模式。
func (a *Authorizer) GetResource(roleID string) ([]string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}

	return role.Resources(), nil
}
