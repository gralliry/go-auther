package model

import (
	"sync"

	"github.com/gralliry/go-auther/internal/resource"
)

// RoleNode represents a node in the role tree.
type RoleNode struct {
	ID       string
	Parent   *RoleNode
	Children map[string]*RoleNode
	Users    map[string]*UserNode

	GrantedMap map[string]bool
	GrantsIn   []*GrantNode
	GrantsOut  []*GrantNode

	matchCache sync.Map
}

// HasAncestor reports whether the given role ID is an ancestor of this role.
func (r *RoleNode) HasAncestor(ancestorID string) bool {
	for p := r; p != nil; p = p.Parent {
		if p.ID == ancestorID {
			return true
		}
	}
	return false
}

// HasResource checks whether any pattern in GrantedMap matches the target resource.
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

// Resources returns all resource patterns in GrantedMap.
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

// GetMatchCache retrieves a cached match result for the given target.
func (r *RoleNode) GetMatchCache(key string) (bool, bool) {
	if r == nil {
		return false, false
	}
	v, ok := r.matchCache.Load(key)
	if !ok {
		return false, false
	}
	return v.(bool), true
}

// SetMatchCache stores a match result in the cache.
func (r *RoleNode) SetMatchCache(key string, val bool) {
	if r == nil {
		return
	}
	r.matchCache.Store(key, val)
}

// ResetMatchCache clears the match cache. Called after write operations.
func (r *RoleNode) ResetMatchCache() {
	if r == nil {
		return
	}
	r.matchCache = sync.Map{}
}

// RoleInfo is the public view of a role, returned by the Authorizer API.
type RoleInfo struct {
	ID         string
	ParentID   string
	Resources  []string
	SubRoleIDs []string
	UserIDs    []string
	GrantsIn   []*GrantNode
	GrantsOut  []*GrantNode
}

// Subtree collects this role and all its descendants using BFS.
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

// ToInfo converts a RoleNode to the public RoleInfo struct.
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
