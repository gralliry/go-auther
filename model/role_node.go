package model

// RoleNode 表示角色树中的一个角色节点。
type RoleNode struct {
	ID         string
	Parent     *RoleNode
	Children   map[string]*RoleNode
	Resources  map[string]bool
	GrantedMap map[string]bool // 索引：精确资源键 → true，O(1) 查找
	GrantsIn   []RoleGrant
	GrantsOut  []RoleGrant
	Users      map[string]*UserNode

	matchCache map[string]bool // 匹配结果缓存（简化版 LRU）
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
