// Package auther provides a role-tree-based authorization library.
//
// # Concepts
//
// Auther manages three core concepts:
//
//   - Role: forms a tree hierarchy. The root role is auto-created with the
//     "/**" resource. Roles can create sub-roles and users.
//     Permissions do NOT auto-inherit — parent roles must explicitly
//     GrantResource to sub-roles.
//
//   - User: a passive leaf created by a role. Users inherit the effective
//     permissions of their creating role but can never manage resources
//     or create other users/roles.
//
//   - Resource: a path-like string (e.g., /user/create, /data/**) supporting
//     glob matching with * (single segment) and ** (zero or more segments).
//
// # Persistence
//
// Adapters provide persistence. Every mutation is immediately written through
// to the adapter (write-through). On construction, state is loaded from the
// adapter if present.
//
// # Thread Safety
//
// Authorizer is safe for concurrent use. All public methods are protected by
// sync.RWMutex.
package auther

import (
	"fmt"
	"sync"
)

// Authorizer is the main entry point for the authorization system.
type Authorizer struct {
	mu        sync.RWMutex
	root      *RoleNode
	roles map[string]*RoleNode
	users map[string]*UserNode
	adapter   Adapter
}

// NewAuthorizer creates a new Authorizer with the given adapter.
// If the adapter has stored data, it is loaded; otherwise a root role
// with ID "root" and resource "/**" is auto-created.
func NewAuthorizer(adapter Adapter) (*Authorizer, error) {
	a := &Authorizer{
		adapter:   adapter,
		roles: make(map[string]*RoleNode),
		users: make(map[string]*UserNode),
	}

	if adapter != nil {
		snap, err := adapter.Load()
		if err != nil {
			return nil, fmt.Errorf("auther: load policy: %w", err)
		}
		if snap != nil && len(snap.Roles) > 0 {
			a.buildTree(snap)
			return a, nil
		}
	}

	a.root = &RoleNode{
		ID:        "root",
		Children:  make(map[string]*RoleNode),
		Resources: map[string]bool{"/**": true},
		Users:     make(map[string]*UserNode),
	}
	a.roles["root"] = a.root

	if adapter != nil {
		if err := a.save(); err != nil {
			return nil, fmt.Errorf("auther: save initial state: %w", err)
		}
	}
	return a, nil
}

// =============================================================================
// Tree helpers
// =============================================================================

func (a *Authorizer) buildTree(snapshot *PolicySnapshot) {
	a.roles = make(map[string]*RoleNode)
	a.users = make(map[string]*UserNode)

	// Phase 1: Create all role nodes.
	for _, rs := range snapshot.Roles {
		role := &RoleNode{
			ID:        rs.ID,
			Children:  make(map[string]*RoleNode),
			Resources: make(map[string]bool),
			Users:     make(map[string]*UserNode),
		}
		for _, res := range rs.Resources {
			role.Resources[res] = true
		}
		a.roles[rs.ID] = role
	}

	// Phase 2: Identify root — first role with empty ParentID wins.
	// If none exists, auto-create a root with "/**".
	var rootID string
	for _, rs := range snapshot.Roles {
		if rs.ParentID == "" {
			rootID = rs.ID
			break
		}
	}
	if rootID == "" {
		rootID = "root"
		if a.roles["root"] == nil {
			a.roles["root"] = &RoleNode{
				ID:        "root",
				Children:  make(map[string]*RoleNode),
				Resources: map[string]bool{"/**": true},
				Users:     make(map[string]*UserNode),
			}
		}
	}
	a.root = a.roles[rootID]

	// Phase 3: Link parents. Orphan roles (bad ParentID) → reattach to root.
	// Extra root candidates (multiple roles with empty ParentID) → child of root.
	for _, rs := range snapshot.Roles {
		if rs.ID == rootID {
			continue
		}
		role := a.roles[rs.ID]
		if role == nil {
			continue
		}
		parent := a.roles[rs.ParentID]
		if parent == nil || rs.ParentID == "" {
			// Orphan or extra root → attach to root.
			parent = a.root
		}
		role.Parent = parent
		parent.Children[rs.ID] = role
	}

	// Phase 4: Load users. Dangling RoleID → drop.
	for _, us := range snapshot.Users {
		role := a.roles[us.RoleID]
		if role == nil {
			continue
		}
		user := &UserNode{ID: us.ID, Role: role}
		a.users[us.ID] = user
		role.Users[us.ID] = user
	}

	// Phase 5: Load grants with validation.
	// Invalid grants (dangling From/To, not-ancestor, duplicate) → silently dropped.
	seen := make(map[string]bool)
	for _, gs := range snapshot.Grants {
		fromRole := a.roles[gs.FromRoleID]
		toRole := a.roles[gs.ToRoleID]
		if fromRole == nil || toRole == nil {
			continue // drop: dangling reference
		}
		// Self-grant: merge into role's own Resources.
		if gs.FromRoleID == gs.ToRoleID {
			toRole.Resources[gs.Resource] = true
			continue
		}
		// Validate ancestor constraint.
		if !a.isAncestorOrSelf(gs.FromRoleID, gs.ToRoleID) {
			continue // drop: not a proper ancestor
		}
		key := gs.FromRoleID + "|" + gs.ToRoleID + "|" + gs.Resource
		if seen[key] {
			continue // drop: duplicate
		}
		seen[key] = true

		grant := RoleGrant{FromRoleID: gs.FromRoleID, ToRoleID: gs.ToRoleID, Resource: gs.Resource}
		fromRole.GrantsOut = append(fromRole.GrantsOut, grant)
		toRole.GrantsIn = append(toRole.GrantsIn, grant)
	}

	// Persist the cleansed state (only if we loaded from an adapter).
	if a.adapter != nil {
		_ = a.adapter.Save(a.toSnapshot())
	}
}

func (a *Authorizer) toSnapshot() *PolicySnapshot {
	snap := &PolicySnapshot{}

	var walk func(role *RoleNode)
	walk = func(role *RoleNode) {
		rs := RoleSnapshot{
			ID:        role.ID,
			Resources: make([]string, 0, len(role.Resources)),
		}
		if role.Parent != nil {
			rs.ParentID = role.Parent.ID
		}
		for res := range role.Resources {
			rs.Resources = append(rs.Resources, res)
		}
		snap.Roles = append(snap.Roles, rs)
		for _, user := range role.Users {
			snap.Users = append(snap.Users, UserSnapshot{ID: user.ID, RoleID: user.Role.ID})
		}
		for _, child := range role.Children {
			walk(child)
		}
	}
	walk(a.root)

	seen := make(map[string]bool)
	var collectGrants func(role *RoleNode)
	collectGrants = func(role *RoleNode) {
		for _, g := range role.GrantsOut {
			key := g.FromRoleID + "|" + g.ToRoleID + "|" + g.Resource
			if !seen[key] {
				seen[key] = true
				snap.Grants = append(snap.Grants, GrantSnapshot{
					FromRoleID: g.FromRoleID, ToRoleID: g.ToRoleID, Resource: g.Resource,
				})
			}
		}
		for _, child := range role.Children {
			collectGrants(child)
		}
	}
	collectGrants(a.root)
	return snap
}

func (a *Authorizer) save() error {
	if a.adapter == nil {
		return nil
	}
	return a.adapter.Save(a.toSnapshot())
}

func (a *Authorizer) collectSubtree(roleID string) []*RoleNode {
	role := a.roles[roleID]
	if role == nil {
		return nil
	}
	var result []*RoleNode
	queue := []*RoleNode{role}
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

func (a *Authorizer) isAncestor(ancestorID, descendantID string) bool {
	d := a.roles[descendantID]
	if d == nil {
		return false
	}
	for d != nil {
		if d.ID == ancestorID {
			return true
		}
		d = d.Parent
	}
	return false
}

func (a *Authorizer) isAncestorOrSelf(aID, dID string) bool {
	return aID == dID || a.isAncestor(aID, dID)
}
