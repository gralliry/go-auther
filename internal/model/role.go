package model

// RoleNode 表示角色树中的一个角色节点。
type RoleNode struct {
	ID         string
	Parent     *RoleNode
	Children   map[string]*RoleNode
	GrantedMap map[string]bool
	GrantsIn   []GrantInfo
	GrantsOut  []GrantInfo
	Users      map[string]*UserNode

	matchCache map[string]bool
}

// GrantInfo 表示从祖先角色到子角色的显式资源授权记录。
type GrantInfo struct {
	FromRoleID string
	ToRoleID   string
	Resource   string
}

const maxMatchCacheSize = 64

// GetMatchCache 从匹配缓存中查找已缓存的结果。
func (r *RoleNode) GetMatchCache(key string) (bool, bool) {
	v, ok := r.matchCache[key]
	return v, ok
}

// SetMatchCache 将匹配结果存入缓存。超过容量时清空全部后重新填充。
func (r *RoleNode) SetMatchCache(key string, val bool) {
	if r.matchCache == nil {
		r.matchCache = make(map[string]bool)
	}
	if len(r.matchCache) >= maxMatchCacheSize {
		r.matchCache = make(map[string]bool)
	}
	r.matchCache[key] = val
}

// ResetMatchCache 清空匹配缓存，在写操作后调用。
func (r *RoleNode) ResetMatchCache() {
	r.matchCache = nil
}
