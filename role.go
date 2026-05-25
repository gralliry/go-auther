package auther

import (
	"fmt"

	"auther/model"
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
		Resources:  make(map[string]bool),
		GrantedMap: make(map[string]bool),
		Users:      make(map[string]*model.UserNode),
	}
	a.roles[roleID] = role
	parent.Children[roleID] = role

	return a.save()
}

// DeleteRole 删除指定角色，级联删除其所有子角色及关联用户。
// 涉及已删除角色的授权记录会从幸存角色中清理。
// 根角色不可删除。
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
	subtree := a.subtree(roleID)
	subtreeSet := make(map[string]bool, len(subtree))
	for _, r := range subtree {
		subtreeSet[r.ID] = true
	}

	// 移除子树中所有用户。
	for _, r := range subtree {
		for userID := range r.Users {
			delete(a.users, userID)
		}
	}

	// 清理幸存角色中与已删除角色相关的授权记录。
	a.cleanGrantsExcluding(subtreeSet)

	// 解除父角色引用后删除子树中的所有角色。
	if target.Parent != nil {
		delete(target.Parent.Children, roleID)
	}
	for _, r := range subtree {
		delete(a.roles, r.ID)
	}

	return a.save()
}

// cleanGrantsExcluding 从幸存角色中移除所有关联角色在排除集中的授权记录，
// 并重建幸存角色的 GrantedMap 和匹配缓存。
func (a *Authorizer) cleanGrantsExcluding(excluded map[string]bool) {
	for _, r := range a.roles {
		if excluded[r.ID] {
			continue
		}
		r.GrantsIn = filterByFrom(r.GrantsIn, excluded)
		r.GrantsOut = filterByTo(r.GrantsOut, excluded)
	}
	for _, r := range a.roles {
		if excluded[r.ID] {
			continue
		}
		r.GrantedMap = make(map[string]bool)
		for _, g := range r.GrantsIn {
			if !excluded[g.FromRoleID] {
				r.GrantedMap[g.Resource] = true
			}
		}
		r.ResetMatchCache()
	}
}

// GetRole 返回指定角色的详细信息。
func (a *Authorizer) GetRole(roleID string) (*model.RoleInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}
	return roleToInfo(role), nil
}

// Roles 返回系统中所有角色的信息列表。
func (a *Authorizer) Roles() []*model.RoleInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var result []*model.RoleInfo
	var walk func(role *model.RoleNode)
	walk = func(role *model.RoleNode) {
		result = append(result, roleToInfo(role))
		for _, child := range role.Children {
			walk(child)
		}
	}
	walk(a.root)
	return result
}

// RoleResources 返回角色当前生效的所有资源权限模式。
// 包含角色自身资源和显式的 GrantsIn，不包含自动继承。
func (a *Authorizer) RoleResources(roleID string) ([]string, error) {
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

// roleToInfo 将内部 RoleNode 转换为对外的 RoleInfo 结构。
func roleToInfo(role *model.RoleNode) *model.RoleInfo {
	info := &model.RoleInfo{
		ID:         role.ID,
		ParentID:   "",
		Resources:  make([]string, 0, len(role.Resources)),
		SubRoleIDs: make([]string, 0, len(role.Children)),
		UserIDs:    make([]string, 0, len(role.Users)),
		GrantsIn:   append([]model.GrantInfo(nil), role.GrantsIn...),
		GrantsOut:  append([]model.GrantInfo(nil), role.GrantsOut...),
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

// filterByFrom 过滤掉 FromRoleID 在排除集合中的授权记录。
func filterByFrom(grants []model.GrantInfo, excluded map[string]bool) []model.GrantInfo {
	out := grants[:0]
	for _, g := range grants {
		if excluded[g.FromRoleID] {
			continue
		}
		out = append(out, g)
	}
	return out
}

// filterByTo 过滤掉 ToRoleID 在排除集合中的授权记录。
func filterByTo(grants []model.GrantInfo, excluded map[string]bool) []model.GrantInfo {
	out := grants[:0]
	for _, g := range grants {
		if excluded[g.ToRoleID] {
			continue
		}
		out = append(out, g)
	}
	return out
}
