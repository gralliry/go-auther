package model

import "github.com/gralliry/go-auther/internal/resource"

// GrantNode represents an explicit resource grant from an ancestor role to a descendant.
type GrantNode struct {
	FromRoleID string
	ToRoleID   string
	Resource   resource.Resource
}

// FilterByFrom filters out grants whose FromRoleID is in the excluded set.
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

// FilterByTo filters out grants whose ToRoleID is in the excluded set.
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

// HasGrant reports whether any grant in the slice matches the given resource.
func HasGrant(grants []*GrantNode, resource resource.Resource) bool {
	for _, g := range grants {
		if g.Resource == resource {
			return true
		}
	}
	return false
}

// GrantKey builds a unique dedup key for a grant.
func GrantKey(fromRoleID, toRoleID string, resource resource.Resource) string {
	return fromRoleID + "|" + toRoleID + "|" + string(resource)
}

// DelGrant removes a grant matching the given source role and resource from the slice.
func DelGrant(grants []*GrantNode, fromRoleID string, resource resource.Resource) []*GrantNode {
	for i, g := range grants {
		if g.FromRoleID == fromRoleID && g.Resource == resource {
			return append(grants[:i], grants[i+1:]...)
		}
	}
	return grants
}

// RemoveGrantsAndCleanup removes sub-grants whose source role no longer covers the resource,
// and cleans up the grantee's GrantedMap and cache.
func RemoveGrantsAndCleanup(grants []*GrantNode, roles map[string]*RoleNode) []*GrantNode {
	out := grants[:0]
	for _, g := range grants {
		from := roles[g.FromRoleID]
		if from == nil || !from.HasResource(string(g.Resource)) {
			if grantee := roles[g.ToRoleID]; grantee != nil {
				grantee.GrantsIn = DelGrant(grantee.GrantsIn, g.FromRoleID, g.Resource)
				if !HasGrant(grantee.GrantsIn, g.Resource) {
					delete(grantee.GrantedMap, string(g.Resource))
				}
				grantee.ResetMatchCache()
			}
			continue
		}
		out = append(out, g)
	}
	return out
}
