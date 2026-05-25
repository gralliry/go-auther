package auther

import (
	"fmt"

	"auther/model"
)

// normalizeRes 调用 model.NewResource 并封装错误为 ErrInvalidResource。
func normalizeRes(raw string) (Resource, error) {
	res, err := model.NewResource(raw)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidResource, err)
	}
	return res, nil
}

// Grant 将资源授权给指定角色。
//
// 当 fromRoleID == toRoleID 时，资源直接添加到该角色自身的资源列表中。
// 否则，从祖先角色向子角色进行授权委托，形成一条授权记录。
// 授权方必须是接收方的祖先角色，否则返回 ErrNotAncestor。
func (a *Authorizer) Grant(fromRoleID, toRoleID, resource string) error {
	res, err := normalizeRes(resource)
	if err != nil {
		return err
	}

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

	// 自授权：直接写入角色自身资源，不产生授权记录。
	if fromRoleID == toRoleID {
		if toRole.Resources[res] {
			return nil
		}
		toRole.Resources[res] = true
		toRole.ResetMatchCache()
		return a.save()
	}

	// 查重：同一 From+To+Resource 组合不允许重复存在。
	for _, g := range fromRole.GrantsOut {
		if g.ToRoleID == toRoleID && g.Resource == res {
			return fmt.Errorf("%w: %s -> %s %s", ErrDuplicateGrant, fromRoleID, toRoleID, res)
		}
	}

	grant := model.GrantInfo{FromRoleID: fromRoleID, ToRoleID: toRoleID, Resource: res}
	fromRole.GrantsOut = append(fromRole.GrantsOut, grant)
	toRole.GrantsIn = append(toRole.GrantsIn, grant)
	toRole.GrantedMap[res] = true
	toRole.ResetMatchCache()
	return a.save()
}

// Revoke 撤销一条授权，并级联删除该子树中所有相同资源的子授权。
func (a *Authorizer) Revoke(fromRoleID, toRoleID, resource string) error {
	res, err := normalizeRes(resource)
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.revokeLocked(fromRoleID, toRoleID, res)
}

// revokeLocked 在持有锁的情况下执行撤销逻辑。
func (a *Authorizer) revokeLocked(fromRoleID, toRoleID string, resource Resource) error {
	fromRole := a.roles[fromRoleID]
	toRole := a.roles[toRoleID]
	if fromRole == nil || toRole == nil {
		return fmt.Errorf("%w", ErrGrantNotFound)
	}

	if fromRoleID == toRoleID {
		return a.revokeSelfLocked(toRole, resource)
	}
	return a.revokeDelegatedLocked(fromRole, toRole, resource)
}

// revokeSelfLocked 移除角色自身的资源权限。
func (a *Authorizer) revokeSelfLocked(role *model.RoleNode, resource Resource) error {
	if !role.Resources[resource] {
		return ErrGrantNotFound
	}
	delete(role.Resources, resource)
	role.ResetMatchCache()
	return a.save()
}

// revokeDelegatedLocked 撤销委托授权，并级联清理子树中相同资源的子授权。
func (a *Authorizer) revokeDelegatedLocked(fromRole, toRole *model.RoleNode, resource Resource) error {
	// 双向删除授权记录。
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
	if !hasGrant(toRole.GrantsIn, resource) {
		delete(toRole.GrantedMap, resource)
	}
	toRole.ResetMatchCache()

	// 级联清理：删除子树中所有针对同一资源的子授权。
	subtree := a.subtree(toRole.ID)
	subtreeSet := make(map[string]bool, len(subtree))
	for _, r := range subtree {
		subtreeSet[r.ID] = true
	}
	for _, r := range subtree {
		r.GrantsOut = removeGrants(r.GrantsOut, resource, subtreeSet, a.roles)
		r.ResetMatchCache()
	}
	return a.save()
}

// removeGrants 从授权列表中移除匹配子树集合中目标角色的所有授权。
func removeGrants(grants []model.GrantInfo, resource Resource, subtreeSet map[string]bool, roles map[string]*model.RoleNode) []model.GrantInfo {
	out := grants[:0]
	for _, g := range grants {
		if g.Resource == resource && subtreeSet[g.ToRoleID] {
			if grantee := roles[g.ToRoleID]; grantee != nil {
				grantee.GrantsIn = delGrant(grantee.GrantsIn, g.FromRoleID, resource)
				if !hasGrant(grantee.GrantsIn, resource) {
					delete(grantee.GrantedMap, resource)
				}
				grantee.ResetMatchCache()
			}
			continue
		}
		out = append(out, g)
	}
	return out
}

// hasGrant 检查授权列表中是否存在指定资源的授权。
func hasGrant(grants []model.GrantInfo, resource Resource) bool {
	for _, g := range grants {
		if g.Resource == resource {
			return true
		}
	}
	return false
}

// delGrant 从接收方的 GrantsIn 列表中移除指定来源和资源的授权记录。
func delGrant(grants []model.GrantInfo, fromRoleID string, resource Resource) []model.GrantInfo {
	for i, g := range grants {
		if g.FromRoleID == fromRoleID && g.Resource == resource {
			return append(grants[:i], grants[i+1:]...)
		}
	}
	return grants
}

// GrantsTo 返回指定角色接收到的所有授权记录（副本）。
func (a *Authorizer) GrantsTo(roleID string) ([]model.GrantInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}
	result := make([]model.GrantInfo, len(role.GrantsIn))
	copy(result, role.GrantsIn)
	return result, nil
}

// GrantsFrom 返回指定角色发出的所有授权记录（副本）。
func (a *Authorizer) GrantsFrom(roleID string) ([]model.GrantInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}
	result := make([]model.GrantInfo, len(role.GrantsOut))
	copy(result, role.GrantsOut)
	return result, nil
}

// AllGrants 返回系统中所有唯一的授权记录。
func (a *Authorizer) AllGrants() []model.GrantInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var result []model.GrantInfo
	seen := make(map[string]bool)
	var walk func(role *model.RoleNode)
	walk = func(role *model.RoleNode) {
		for _, g := range role.GrantsOut {
			key := g.FromRoleID + "|" + g.ToRoleID + "|" + string(g.Resource)
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
