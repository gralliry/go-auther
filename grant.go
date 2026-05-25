package auther

import (
	"fmt"

	"auther/model"
)

// GrantResource 将资源授权给指定角色。
//
// 当 fromRoleID == toRoleID 时，资源直接添加到该角色自身的资源列表中。
// 否则，从祖先角色向子角色进行授权委托，形成一条授权记录。
// 授权方必须是接收方的祖先角色，否则返回 ErrNotAncestor。
func (a *Authorizer) GrantResource(fromRoleID, toRoleID, resource string) error {
	var err error
	resource, err = normalizeResource(resource)
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
		if toRole.Resources[resource] {
			return nil
		}
		toRole.Resources[resource] = true
		return a.save()
	}

	// 查重：同一 From+To+Resource 组合不允许重复存在。
	for _, g := range fromRole.GrantsOut {
		if g.ToRoleID == toRoleID && g.Resource == resource {
			return fmt.Errorf("%w: %s -> %s %s", ErrDuplicateGrant, fromRoleID, toRoleID, resource)
		}
	}

	grant := model.RoleGrant{FromRoleID: fromRoleID, ToRoleID: toRoleID, Resource: resource}
	fromRole.GrantsOut = append(fromRole.GrantsOut, grant)
	toRole.GrantsIn = append(toRole.GrantsIn, grant)
	toRole.GrantedMap[resource] = true
	return a.save()
}

// RevokeResource 撤销一条授权，并级联删除该子树中所有相同资源的子授权。
func (a *Authorizer) RevokeResource(fromRoleID, toRoleID, resource string) error {
	var err error
	resource, err = normalizeResource(resource)
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.revokeResourceLocked(fromRoleID, toRoleID, resource)
}

// revokeResourceLocked 在持有锁的情况下执行撤销逻辑。
func (a *Authorizer) revokeResourceLocked(fromRoleID, toRoleID, resource string) error {
	fromRole := a.roles[fromRoleID]
	toRole := a.roles[toRoleID]
	if fromRole == nil || toRole == nil {
		return fmt.Errorf("%w", ErrGrantNotFound)
	}

	// 自撤销：从角色自身资源中移除。
	if fromRoleID == toRoleID {
		if !toRole.Resources[resource] {
			return ErrGrantNotFound
		}
		delete(toRole.Resources, resource)
		return a.save()
	}

	// 从授权方和接收方的授权列表中双向删除该条授权记录。
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
	// 检查是否还有同资源的其他授权
	stillHas := false
	for _, g := range toRole.GrantsIn {
		if g.Resource == resource {
			stillHas = true
			break
		}
	}
	if !stillHas {
		delete(toRole.GrantedMap, resource)
	}
	if !found {
		return fmt.Errorf("%w: %s -> %s %s", ErrGrantNotFound, fromRoleID, toRoleID, resource)
	}

	// 级联清理：删除子树中所有针对同一资源的子授权。
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

// removeSubtreeGrants 从授权列表中移除匹配子树集合中目标角色的所有授权。
func removeSubtreeGrants(grants []model.RoleGrant, resource string, subtreeSet map[string]bool, roles map[string]*model.RoleNode) []model.RoleGrant {
	out := grants[:0]
	for _, g := range grants {
		if g.Resource == resource && subtreeSet[g.ToRoleID] {
			if grantee := roles[g.ToRoleID]; grantee != nil {
				grantee.GrantsIn = removeGrantIn(grantee.GrantsIn, g.FromRoleID, resource)
				// 检查是否还有同资源的其他授权
				stillHas := false
				for _, gr := range grantee.GrantsIn {
					if gr.Resource == resource {
						stillHas = true
						break
					}
				}
				if !stillHas {
					delete(grantee.GrantedMap, resource)
				}
			}
			continue
		}
		out = append(out, g)
	}
	return out
}

// removeGrantIn 从接收方的 GrantsIn 列表中移除指定来源和资源的授权记录。
func removeGrantIn(grants []model.RoleGrant, fromRoleID, resource string) []model.RoleGrant {
	for i, g := range grants {
		if g.FromRoleID == fromRoleID && g.Resource == resource {
			return append(grants[:i], grants[i+1:]...)
		}
	}
	return grants
}

// GetGrantsToRole 返回指定角色接收到的所有授权记录（副本）。
func (a *Authorizer) GetGrantsToRole(roleID string) ([]model.RoleGrant, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}
	result := make([]model.RoleGrant, len(role.GrantsIn))
	copy(result, role.GrantsIn)
	return result, nil
}

// GetGrantsFromRole 返回指定角色发出的所有授权记录（副本）。
func (a *Authorizer) GetGrantsFromRole(roleID string) ([]model.RoleGrant, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	role := a.roles[roleID]
	if role == nil {
		return nil, fmt.Errorf("%w: %s", ErrRoleNotFound, roleID)
	}
	result := make([]model.RoleGrant, len(role.GrantsOut))
	copy(result, role.GrantsOut)
	return result, nil
}

// GetAllGrants 返回系统中所有唯一的授权记录。
func (a *Authorizer) GetAllGrants() []model.RoleGrant {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var result []model.RoleGrant
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
