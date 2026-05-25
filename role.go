package auther

import (
	"fmt"

	"github.com/gralliry/go-auther/internal/model"
)

// RoleInfo 是对外暴露的角色信息视图。
type RoleInfo struct {
	ID         string
	ParentID   string
	Resources  []string
	SubRoleIDs []string
	UserIDs    []string
	GrantsIn   []*GrantInfo
	GrantsOut  []*GrantInfo
}

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

	return a.saveSetRole(roleID, parentID)
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
	subtree := a.subtree(roleID)
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
	a.cleanGrantsExcluding(excluded)

	// 解除父角色引用后删除子树中的所有角色。
	if target.Parent != nil {
		delete(target.Parent.Children, roleID)
	}
	for _, r := range subtree {
		delete(a.roles, r.ID)
	}

	return a.save()
}

// cleanGrantsExcluding 从幸存角色中移除关联已删除角色的授权记录，并重建 GrantedMap。
func (a *Authorizer) cleanGrantsExcluding(excluded map[string]bool) {
	for _, r := range a.roles {
		if excluded[r.ID] {
			continue
		}
		r.GrantsIn = filterByFrom(r.GrantsIn, excluded)
		r.GrantsOut = filterByTo(r.GrantsOut, excluded)
		r.GrantedMap = make(map[string]bool)
		for _, g := range r.GrantsIn {
			r.GrantedMap[g.Resource] = true
		}
		r.ResetMatchCache()
	}
}

// GetRole 返回指定角色的详细信息。
func (a *Authorizer) GetRole(roleID string) (*RoleInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}
	return roleToInfo(role), nil
}

// GetAllRoles 返回系统中所有角色的信息列表。
func (a *Authorizer) GetAllRoles() []*RoleInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var result []*RoleInfo
	var walk func(role *model.RoleNode)
	walk = func(role *model.RoleNode) {
		result = append(result, roleToInfo(role))
		for _, child := range role.Children {
			walk(child)
		}
	}
	if root := a.roles["root"]; root != nil {
		walk(root)
	}
	return result
}

// GetSubRoles 返回指定角色的直接子角色列表。
func (a *Authorizer) GetSubRoles(roleID string) ([]*RoleInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}

	result := make([]*RoleInfo, 0, len(role.Children))
	for _, child := range role.Children {
		result = append(result, roleToInfo(child))
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

	seen := make(map[string]bool)
	var result []string
	for r := range role.GrantedMap {
		if !seen[r] {
			seen[r] = true
			result = append(result, r)
		}
	}
	return result, nil
}

// roleToInfo 将内部 RoleNode 转换为对外的 RoleInfo 结构。
func roleToInfo(role *model.RoleNode) *RoleInfo {
	info := &RoleInfo{
		ID:         role.ID,
		Resources:  make([]string, 0, len(role.GrantsIn)),
		SubRoleIDs: make([]string, 0, len(role.Children)),
		UserIDs:    make([]string, 0, len(role.Users)),
		GrantsIn:   append([]*model.GrantInfo(nil), role.GrantsIn...),
		GrantsOut:  append([]*model.GrantInfo(nil), role.GrantsOut...),
	}
	if role.Parent != nil {
		info.ParentID = role.Parent.ID
	}
	for r := range role.GrantedMap {
		info.Resources = append(info.Resources, r)
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
func filterByFrom(grants []*model.GrantInfo, excluded map[string]bool) []*model.GrantInfo {
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
func filterByTo(grants []*model.GrantInfo, excluded map[string]bool) []*model.GrantInfo {
	out := grants[:0]
	for _, g := range grants {
		if excluded[g.ToRoleID] {
			continue
		}
		out = append(out, g)
	}
	return out
}
