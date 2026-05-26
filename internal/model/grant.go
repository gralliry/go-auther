package model

// GrantNode 表示从祖先角色到子角色的显式资源授权记录。
type GrantNode struct {
	FromRoleID string
	ToRoleID   string
	Resource   string
}

// FilterByFrom 过滤掉 FromRoleID 在排除集合中的授权记录。
func FilterByFrom(grants []*GrantNode, excluded map[string]bool) []*GrantNode {
	out := grants[:0]
	for _, g := range grants {
		if excluded[g.FromRoleID] {
			continue
		}
		out = append(out, g)
	}
	return out
}

// FilterByTo 过滤掉 ToRoleID 在排除集合中的授权记录。
func FilterByTo(grants []*GrantNode, excluded map[string]bool) []*GrantNode {
	out := grants[:0]
	for _, g := range grants {
		if excluded[g.ToRoleID] {
			continue
		}
		out = append(out, g)
	}
	return out
}

// HasGrant 检查授权列表中是否存在指定资源的授权。
func HasGrant(grants []*GrantNode, resource string) bool {
	for _, g := range grants {
		if g.Resource == resource {
			return true
		}
	}
	return false
}

// GrantKey 构建授权的唯一标识键，用于去重。
func GrantKey(fromRoleID, toRoleID, resource string) string {
	return fromRoleID + "|" + toRoleID + "|" + resource
}

// DelGrant 从授权列表中移除指定来源和资源的授权记录。
func DelGrant(grants []*GrantNode, fromRoleID string, resource string) []*GrantNode {
	for i, g := range grants {
		if g.FromRoleID == fromRoleID && g.Resource == resource {
			return append(grants[:i], grants[i+1:]...)
		}
	}
	return grants
}

// RemoveGrantsAndCleanup 移除已失去覆盖的转授记录，并清理受让角色的 GrantedMap 和缓存。
func RemoveGrantsAndCleanup(grants []*GrantNode, roles map[string]*RoleNode) []*GrantNode {
	out := grants[:0]
	for _, g := range grants {
		from := roles[g.FromRoleID]
		if from == nil || !from.HasResource(g.Resource) {
			if grantee := roles[g.ToRoleID]; grantee != nil {
				grantee.GrantsIn = DelGrant(grantee.GrantsIn, g.FromRoleID, g.Resource)
				if !HasGrant(grantee.GrantsIn, g.Resource) {
					delete(grantee.GrantedMap, g.Resource)
				}
				grantee.ResetMatchCache()
			}
			continue
		}
		out = append(out, g)
	}
	return out
}
