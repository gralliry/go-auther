package model

import "github.com/gralliry/go-auther/internal/resource"

// RoleNode 表示角色树中的一个角色节点。
type RoleNode struct {
	ID       string
	Parent   *RoleNode
	Children map[string]*RoleNode
	Users    map[string]*UserNode

	GrantedMap map[string]bool
	GrantsIn   []*GrantNode
	GrantsOut  []*GrantNode

	matchCache map[string]bool
}

const maxMatchCacheSize = 64

// HasAncestor 判断给定角色ID是否为当前角色的祖先。
func (r *RoleNode) HasAncestor(ancestorID string) bool {
	for p := r; p != nil; p = p.Parent {
		if p.ID == ancestorID {
			return true
		}
	}
	return false
}

// HasResource 检查 GrantedMap 中是否有模式能匹配目标资源。
func (r *RoleNode) HasResource(target string) bool {
	if r == nil {
		return false
	}
	if r.GrantedMap[target] {
		return true
	}
	if cached, ok := r.GetMatchCache(target); ok {
		return cached
	}
	for pattern := range r.GrantedMap {
		if resource.Resource(pattern).Match(target) {
			r.SetMatchCache(target, true)
			return true
		}
	}
	r.SetMatchCache(target, false)
	return false
}

// Resources 返回 GrantedMap 中所有资源模式。
func (r *RoleNode) Resources() []string {
	if r == nil {
		return nil
	}
	result := make([]string, 0, len(r.GrantedMap))
	for res := range r.GrantedMap {
		result = append(result, res)
	}
	return result
}

// GetMatchCache 从匹配缓存中查找已缓存的结果。
func (r *RoleNode) GetMatchCache(key string) (bool, bool) {
	if r == nil {
		return false, false
	}
	v, ok := r.matchCache[key]
	return v, ok
}

// SetMatchCache 将匹配结果存入缓存。超过容量时清空全部后重新填充。
func (r *RoleNode) SetMatchCache(key string, val bool) {
	if r == nil {
		return
	}
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
	if r == nil {
		return
	}
	r.matchCache = nil
}

// RoleInfo 是对外暴露的角色信息视图。
type RoleInfo struct {
	ID         string
	ParentID   string
	Resources  []string
	SubRoleIDs []string
	UserIDs    []string
	GrantsIn   []*GrantNode
	GrantsOut  []*GrantNode
}

// Subtree 收集当前角色及其所有后代角色，使用 BFS 遍历。
func (r *RoleNode) Subtree() []*RoleNode {
	if r == nil {
		return nil
	}
	var result []*RoleNode
	queue := []*RoleNode{r}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		result = append(result, cur)
		for _, child := range cur.Children {
			queue = append(queue, child)
		}
	}
	return result
}

// ToInfo 将 RoleNode 转换为对外的 RoleInfo 结构。
func (r *RoleNode) ToInfo() *RoleInfo {
	if r == nil {
		return nil
	}
	info := &RoleInfo{
		ID:         r.ID,
		Resources:  r.Resources(),
		SubRoleIDs: make([]string, 0, len(r.Children)),
		UserIDs:    make([]string, 0, len(r.Users)),
		GrantsIn:   append([]*GrantNode(nil), r.GrantsIn...),
		GrantsOut:  append([]*GrantNode(nil), r.GrantsOut...),
	}
	if r.Parent != nil {
		info.ParentID = r.Parent.ID
	}
	for childID := range r.Children {
		info.SubRoleIDs = append(info.SubRoleIDs, childID)
	}
	for userID := range r.Users {
		info.UserIDs = append(info.UserIDs, userID)
	}
	return info
}
