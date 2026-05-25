package auther

import (
	"fmt"

	"github.com/gralliry/go-auther/internal/match"

	"github.com/gralliry/go-auther/internal/model"
)

// normalizeRes 调用 match.Clean 并封装错误为 ErrInvalidResource。
func normalizeRes(raw string) (string, error) {
	res, err := match.Clean(raw)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidResource, err)
	}
	return res, nil
}

// Grant 将资源从祖先角色授权给后代角色。
//
// 授权方必须是接收方的祖先角色（不允许自授权），否则返回 ErrNotAncestor。
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

	if !a.isAncestor(fromRoleID, toRoleID) {
		return fmt.Errorf("%w: %s is not an ancestor of %s", ErrNotAncestor, fromRoleID, toRoleID)
	}

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
	return a.saveSetGrant(fromRoleID, toRoleID, res)
}

// Revoke 撤销一条授权，并级联删除该子树中所有相同资源的子授权。
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

// revokeDelegatedLocked 撤销委托授权，并级联清理子树中相同资源的子授权。
func (a *Authorizer) revokeDelegatedLocked(fromRole, toRole *model.RoleNode, resource string) error {
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
func removeGrants(grants []model.GrantInfo, resource string, subtreeSet map[string]bool, roles map[string]*model.RoleNode) []model.GrantInfo {
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
func hasGrant(grants []model.GrantInfo, resource string) bool {
	for _, g := range grants {
		if g.Resource == resource {
			return true
		}
	}
	return false
}

// delGrant 从接收方的 GrantsIn 列表中移除指定来源和资源的授权记录。
func delGrant(grants []model.GrantInfo, fromRoleID string, resource string) []model.GrantInfo {
	for i, g := range grants {
		if g.FromRoleID == fromRoleID && g.Resource == resource {
			return append(grants[:i], grants[i+1:]...)
		}
	}
	return grants
}

// GetGrantsTo 返回指定角色接收到的授权记录。
func (a *Authorizer) GetGrantsTo(roleID string) ([]model.GrantInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}
	return append([]model.GrantInfo(nil), role.GrantsIn...), nil
}

// GetGrantsFrom 返回指定角色发出的授权记录。
func (a *Authorizer) GetGrantsFrom(roleID string) ([]model.GrantInfo, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}
	return append([]model.GrantInfo(nil), role.GrantsOut...), nil
}

// GetAllGrants 返回系统中所有唯一的授权记录。
func (a *Authorizer) GetAllGrants() []model.GrantInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var result []model.GrantInfo
	seen := make(map[string]bool)
	var walk func(role *model.RoleNode)
	walk = func(role *model.RoleNode) {
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
