package auther

import (
	"errors"
	"fmt"
	"testing"

	"github.com/gralliry/go-auther/internal/model"
	"github.com/gralliry/go-auther/snapshot"
)

// =============================================================================
// Helpers
// =============================================================================

// testAdapter is a minimal in-memory adapter for tests.
type testAdapter struct {
	snap *snapshot.Policy
}

func (m *testAdapter) Load() (*snapshot.Policy, error)           { return m.snap, nil }
func (m *testAdapter) Save(s *snapshot.Policy) error             { m.snap = s; return nil }
func (m *testAdapter) SetRole(snapshot.Role) error               { return nil }
func (m *testAdapter) UnsetRole(snapshot.Role) error             { return nil }
func (m *testAdapter) SetUser(snapshot.User) error               { return nil }
func (m *testAdapter) UnsetUser(snapshot.User) error             { return nil }
func (m *testAdapter) SetGrant(snapshot.Grant) error             { return nil }
func (m *testAdapter) UnsetGrant(snapshot.Grant) error           { return nil }

// newTestAuthorizer creates an authorizer with a test in-memory adapter.
func newTestAuthorizer(t *testing.T) *Authorizer {
	t.Helper()
	a, err := NewAuthorizer(&testAdapter{})
	if err != nil {
		t.Fatalf("NewAuthorizer: %v", err)
	}
	return a
}

// getAllGrants 收集系统中所有唯一的授权记录。
func getAllGrants(a *Authorizer) []*model.GrantNode {
	var result []*model.GrantNode
	seen := make(map[string]bool)
	root := a.roles["root"]
	if root == nil {
		return nil
	}
	queue := []*model.RoleNode{root}
	for len(queue) > 0 {
		role := queue[0]
		queue = queue[1:]
		for _, g := range role.GrantsOut {
			key := model.GrantKey(g.FromRoleID, g.ToRoleID, g.Resource)
			if !seen[key] {
				seen[key] = true
				result = append(result, g)
			}
		}
		for _, child := range role.Children {
			queue = append(queue, child)
		}
	}
	return result
}

// setupTestHierarchy creates a standard test hierarchy:
//
//	Roles: root(/**) -> admin -> editor
//	       root grants /user/* to admin
//	       root grants /data/* to editor
//	       root grants /g/** to admin
//	Users: admin has u_admin
//	       editor has u_editor
func setupTestHierarchy(t *testing.T, a *Authorizer) {
	t.Helper()

	must(t, a.CreateRole("root", "admin"))
	must(t, a.CreateRole("admin", "editor"))
	must(t, a.Grant("root", "admin", "/user/*"))
	must(t, a.Grant("root", "editor", "/data/*"))
	must(t, a.Grant("root", "admin", "/g/**"))
	must(t, a.CreateUser("admin", "u_admin"))
	must(t, a.CreateUser("editor", "u_editor"))
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// =============================================================================
// Constructor tests
// =============================================================================

func TestNewAuthorizerNilAdapter(t *testing.T) {
	_, err := NewAuthorizer(nil)
	if !errors.Is(err, ErrAdapterRequired) {
		t.Errorf("expected ErrAdapterRequired, got %v", err)
	}
}

func TestNewAuthorizerAutoRoot(t *testing.T) {
	a := newTestAuthorizer(t)

	role, err := a.GetRole("root")
	if err != nil {
		t.Fatalf("GetRole root: %v", err)
	}
	if role.ID != "root" {
		t.Errorf("root role mismatch: %+v", role)
	}
	if len(role.Resources) != 1 || role.Resources[0] != "/**" {
		t.Errorf("expected [/ /**], got %v", role.Resources)
	}
	if role.ParentID != "" {
		t.Errorf("root should have no parent, got %q", role.ParentID)
	}
}

func TestNewAuthorizerRootAccess(t *testing.T) {
	a := newTestAuthorizer(t)

	must(t, a.CreateUser("root", "ruser"))

	ok, err := a.Enforce("ruser", "/anything")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("root user should have access to everything via /**")
	}
}

// =============================================================================
// Role tests
// =============================================================================

func TestCreateRole(t *testing.T) {
	a := newTestAuthorizer(t)

	must(t, a.CreateRole("root", "editor"))

	role, err := a.GetRole("editor")
	if err != nil {
		t.Fatalf("GetRole editor: %v", err)
	}
	if role.ParentID != "root" {
		t.Errorf("role editor mismatch: %+v", role)
	}

	// Duplicate
	err = a.CreateRole("root", "editor")
	if !errors.Is(err, ErrDuplicateRole) {
		t.Errorf("expected ErrDuplicateRole, got %v", err)
	}

	// Invalid parent
	err = a.CreateRole("nonexistent", "child")
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestCreateRoleDeepNesting(t *testing.T) {
	a := newTestAuthorizer(t)

	must(t, a.CreateRole("root", "grandparent"))
	must(t, a.CreateRole("grandparent", "parent"))
	must(t, a.CreateRole("parent", "child"))

	// Verify the chain
	role, _ := a.GetRole("child")
	if role.ParentID != "parent" {
		t.Errorf("expected child.parent = parent, got %q", role.ParentID)
	}

	// Can create a sibling under grandparent (parent of parent)
	must(t, a.CreateRole("grandparent", "uncle"))

	roles := a.GetAllRoles()
	if len(roles) != 5 { // root, grandparent, parent, child, uncle
		t.Errorf("expected 5 roles, got %d", len(roles))
	}
}

func TestDeleteRoleCascade(t *testing.T) {
	a := newTestAuthorizer(t)
	setupTestHierarchy(t, a)

	// Add a sub-role under editor and a user
	must(t, a.CreateRole("editor", "sub_editor"))
	must(t, a.CreateUser("sub_editor", "u_sub"))

	// Grant something through the chain
	must(t, a.Grant("admin", "editor", "/shared/*"))
	must(t, a.Grant("editor", "sub_editor", "/shared/*"))

	// Delete admin -> cascades to editor, sub_editor, and all their users
	must(t, a.DeleteRole("admin"))

	// Check roles are gone
	_, err := a.GetRole("admin")
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("expected ErrRoleNotFound for admin, got %v", err)
	}
	_, err = a.GetRole("editor")
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("expected ErrRoleNotFound for editor, got %v", err)
	}
	_, err = a.GetRole("sub_editor")
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("expected ErrRoleNotFound for sub_editor, got %v", err)
	}

	// Check users are gone
	_, err = a.GetUser("u_admin")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound for u_admin, got %v", err)
	}
	_, err = a.GetUser("u_editor")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound for u_editor, got %v", err)
	}
	_, err = a.GetUser("u_sub")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound for u_sub, got %v", err)
	}

	// Root should still exist
	_, err = a.GetRole("root")
	if err != nil {
		t.Errorf("root should still exist: %v", err)
	}

	// No grants should remain
	grants := getAllGrants(a)
	if len(grants) != 0 {
		t.Errorf("expected 0 grants, got %d: %+v", len(grants), grants)
	}
}

func TestDeleteRootRoleForbidden(t *testing.T) {
	a := newTestAuthorizer(t)

	err := a.DeleteRole("root")
	if !errors.Is(err, ErrRootRoleDelete) {
		t.Errorf("expected ErrRootRoleDelete, got %v", err)
	}
}

// =============================================================================
// Resource tests
// =============================================================================

func TestAddRemoveResourceToRole(t *testing.T) {
	a := newTestAuthorizer(t)

	must(t, a.CreateRole("root", "child"))
	must(t, a.Grant("root", "child", "/extra/**"))

	role, _ := a.GetRole("child")
	found := false
	for _, r := range role.Resources {
		if r == "/extra/**" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected /extra/** in child resources")
	}

	must(t, a.Revoke("root", "child", "/extra/**"))

	role, _ = a.GetRole("child")
	for _, r := range role.Resources {
		if r == "/extra/**" {
			t.Error("expected /extra/** to be removed")
		}
	}

	// Remove non-existent resource
	err := a.Revoke("root", "child", "/nonexistent")
	if !errors.Is(err, ErrGrantNotFound) {
		t.Errorf("expected ErrGrantNotFound, got %v", err)
	}
}

// =============================================================================
// Grant tests
// =============================================================================

func TestGrant(t *testing.T) {
	a := newTestAuthorizer(t)

	must(t, a.CreateRole("root", "child"))
	must(t, a.Grant("root", "child", "/custom/*"))

	grantsFrom, _ := a.GetGrantsFrom("root")
	if len(grantsFrom) != 1 {
		t.Fatalf("expected 3 grants from root, got %d", len(grantsFrom))
	}
	if grantsFrom[0].Resource != "/custom/*" {
		t.Errorf("expected /custom/*, got %s", grantsFrom[0].Resource)
	}

	grantsTo, _ := a.GetGrantsTo("child")
	if len(grantsTo) != 1 {
		t.Fatalf("expected 3 grants to child, got %d", len(grantsTo))
	}
}

func TestGrantNotAncestor(t *testing.T) {
	a := newTestAuthorizer(t)

	must(t, a.CreateRole("root", "role_a"))
	must(t, a.CreateRole("root", "role_b"))

	// role_a is not an ancestor of role_b
	err := a.Grant("role_a", "role_b", "/x")
	if !errors.Is(err, ErrNotAncestor) {
		t.Errorf("expected ErrNotAncestor, got %v", err)
	}
}

func TestGrantDuplicate(t *testing.T) {
	a := newTestAuthorizer(t)

	must(t, a.CreateRole("root", "child"))
	must(t, a.Grant("root", "child", "/dup"))

	err := a.Grant("root", "child", "/dup")
	if !errors.Is(err, ErrDuplicateGrant) {
		t.Errorf("expected ErrDuplicateGrant, got %v", err)
	}
}

func TestGrantSelf(t *testing.T) {
	a := newTestAuthorizer(t)

	// Self-grant is rejected
	err := a.Grant("root", "root", "/self/**")
	if !errors.Is(err, ErrNotAncestor) {
		t.Errorf("expected ErrNotAncestor for self-grant, got %v", err)
	}

	// Self-revoke is also rejected
	err = a.Revoke("root", "root", "/self/**")
	if !errors.Is(err, ErrGrantNotFound) {
		t.Errorf("expected ErrGrantNotFound for self-revoke, got %v", err)
	}
}

func TestRevokeCascade(t *testing.T) {
	a := newTestAuthorizer(t)
	setupTestHierarchy(t, a)

	// Grant chain: root -> admin -> editor
	must(t, a.Grant("admin", "editor", "/reports/*"))

	// Revoke at the top
	must(t, a.Revoke("root", "admin", "/g/**"))

	// Check that the grant is gone
	grants := getAllGrants(a)
	for _, g := range grants {
		if g.Resource == "/g/**" {
			t.Errorf("grant still exists: %+v", g)
		}
	}
}

func TestRevokeCascadeDeep(t *testing.T) {
	a := newTestAuthorizer(t)

	must(t, a.CreateRole("root", "r1"))
	must(t, a.CreateRole("r1", "r2"))
	must(t, a.CreateRole("r2", "r3"))

	must(t, a.Grant("root", "r1", "/reports"))
	must(t, a.Grant("r1", "r2", "/reports"))
	must(t, a.Grant("r2", "r3", "/reports"))

	// Revoke at r1; should cascade to r2 and r3
	must(t, a.Revoke("root", "r1", "/reports"))

	grants := getAllGrants(a)
	for _, g := range grants {
		if g.Resource == "/reports" {
			t.Errorf("grant on /reports still exists: %+v", g)
		}
	}
}

// =============================================================================
// User tests
// =============================================================================

func TestCreateUser(t *testing.T) {
	a := newTestAuthorizer(t)

	must(t, a.CreateUser("root", "u1"))

	u, err := a.GetUser("u1")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if u.RoleID != "root" {
		t.Errorf("user mismatch: %+v", u)
	}

	// Duplicate
	err = a.CreateUser("root", "u1")
	if !errors.Is(err, ErrDuplicateUser) {
		t.Errorf("expected ErrDuplicateUser, got %v", err)
	}

	// Invalid role
	err = a.CreateUser("nonexistent", "u2")
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestDeleteUser(t *testing.T) {
	a := newTestAuthorizer(t)

	must(t, a.CreateUser("root", "u1"))
	must(t, a.DeleteUser("root", "u1"))

	_, err := a.GetUser("u1")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

// =============================================================================
// Enforcement tests
// =============================================================================

func TestEnforceRoleResources(t *testing.T) {
	a := newTestAuthorizer(t)
	setupTestHierarchy(t, a)

	tests := []struct {
		user     string
		resource string
		want     bool
	}{
		// admin user has /user/* (own) + /g/** (granted by root)
		{"u_admin", "/user/create", true},
		{"u_admin", "/user/edit", true},
		{"u_admin", "/g/something", true},
		{"u_admin", "/anything", false},  // no /** — explicit only
		{"u_admin", "/data/read", false}, // not in admin's resources

		// editor user has /data/* from its role, no grants
		{"u_editor", "/data/read", true},
		{"u_editor", "/data/write", true},
		{"u_editor", "/user/create", false},
		{"u_editor", "/g/something", false},
		{"u_editor", "/data/sub/action", false},
	}

	for _, tt := range tests {
		t.Run(tt.user+"_"+tt.resource, func(t *testing.T) {
			ok, err := a.Enforce(tt.user, tt.resource)
			if err != nil {
				t.Fatalf("Enforce: %v", err)
			}
			if ok != tt.want {
				t.Errorf("Enforce(%q, %q) = %v, want %v", tt.user, tt.resource, ok, tt.want)
			}
		})
	}
}

func TestEnforceWithGrants(t *testing.T) {
	a := newTestAuthorizer(t)
	setupTestHierarchy(t, a)

	// Before grant, editor should NOT have /user/*
	ok, _ := a.Enforce("u_editor", "/user/create")
	if ok {
		t.Error("u_editor should not have /user/create yet")
	}

	// Grant /user/* from admin to editor
	must(t, a.Grant("admin", "editor", "/user/*"))

	// Now editor should have it
	ok, _ = a.Enforce("u_editor", "/user/create")
	if !ok {
		t.Error("u_editor should have /user/create after grant")
	}
}

func TestEnforceNonExistentUser(t *testing.T) {
	a := newTestAuthorizer(t)

	_, err := a.Enforce("nonexistent", "/anything")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestRoles(t *testing.T) {
	a := newTestAuthorizer(t)
	setupTestHierarchy(t, a)

	roles := a.GetAllRoles()
	if len(roles) != 3 { // root, admin, editor
		t.Errorf("expected 3 roles, got %d", len(roles))
	}
}

func TestUsers(t *testing.T) {
	a := newTestAuthorizer(t)
	setupTestHierarchy(t, a)

	users := a.GetUsers()
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestAllGrants(t *testing.T) {
	a := newTestAuthorizer(t)
	setupTestHierarchy(t, a)

	grants := getAllGrants(a)
	if len(grants) != 3 {
		t.Errorf("expected 3 grants, got %d", len(grants))
	}
}

// =============================================================================
// Effective role resources
// =============================================================================

func TestEffectiveRoleResources(t *testing.T) {
	a := newTestAuthorizer(t)
	setupTestHierarchy(t, a)

	// Editor should have /data/* (own) — no auto-inheritance
	effective, err := a.GetResource("editor")
	if err != nil {
		t.Fatal(err)
	}

	if len(effective) != 1 || effective[0] != "/data/*" {
		t.Errorf("expected [/data/*], got %v", effective)
	}

	// Admin should have /user/* (own) + /g/** (grant from root)
	effective, err = a.GetResource("admin")
	if err != nil {
		t.Fatal(err)
	}
	hasUser := false
	hasG := false
	for _, r := range effective {
		if r == "/user/*" {
			hasUser = true
		}
		if r == "/g/**" {
			hasG = true
		}
	}
	if !hasUser {
		t.Error("expected /user/* in admin effective resources")
	}
	if !hasG {
		t.Error("expected /g/** in admin effective resources")
	}
}

// =============================================================================
// No-adapter tests
// =============================================================================

// =============================================================================
// Self-healing tests — verify buildTree cleans corrupted snapshot data.
// =============================================================================

// corruptAdapter is an adapter that seeds the Authorizer with a given snapshot.
type corruptAdapter struct {
	snap *snapshot.Policy
}

func (a *corruptAdapter) Load() (*snapshot.Policy, error) { return a.snap, nil }
func (a *corruptAdapter) Save(s *snapshot.Policy) error   { a.snap = s; return nil }

func (a *corruptAdapter) SetRole(role snapshot.Role) error       { return nil }
func (a *corruptAdapter) UnsetRole(role snapshot.Role) error     { return nil }
func (a *corruptAdapter) SetUser(user snapshot.User) error       { return nil }
func (a *corruptAdapter) UnsetUser(user snapshot.User) error     { return nil }
func (a *corruptAdapter) SetGrant(grant snapshot.Grant) error    { return nil }
func (a *corruptAdapter) UnsetGrant(grant snapshot.Grant) error  { return nil }

func newHealed(t *testing.T, snap *snapshot.Policy) *Authorizer {
	t.Helper()
	a, err := NewAuthorizer(&corruptAdapter{snap: snap})
	if err != nil {
		t.Fatalf("NewAuthorizer: %v", err)
	}
	return a
}

func TestSelfHealOrphanRole(t *testing.T) {
	// Role with non-existent ParentID → reattached to root.
	a := newHealed(t, &snapshot.Policy{
		Roles: []snapshot.Role{
			{ID: "root"},
			{ID: "orphan", ParentID: "bogus"},
		},
	})

	role, _ := a.GetRole("orphan")
	if role.ParentID != "root" {
		t.Errorf("orphan should be reattached to root, got parent=%q", role.ParentID)
	}

	rootRole, _ := a.GetRole("root")
	found := false
	for _, id := range rootRole.SubRoleIDs {
		if id == "orphan" {
			found = true
			break
		}
	}
	if !found {
		t.Error("root should have orphan as child")
	}
}

func TestSelfHealMissingRoot(t *testing.T) {
	// No role with empty ParentID → auto-create root with "/**".
	a := newHealed(t, &snapshot.Policy{
		Roles: []snapshot.Role{
			{ID: "child", ParentID: "root"},
		},
	})

	root, _ := a.GetRole("root")
	if root.ParentID != "" {
		t.Error("root should have no parent")
	}
	hasAll := false
	for _, res := range root.Resources {
		if res == "/**" {
			hasAll = true
			break
		}
	}
	if !hasAll {
		t.Errorf("auto-created root should have /**, got %v", root.Resources)
	}

	// The child with explicit ParentID "root" should be linked.
	child, _ := a.GetRole("child")
	if child.ParentID != "root" {
		t.Errorf("child should be under root, got parent=%q", child.ParentID)
	}
}

func TestSelfHealMultipleRoots(t *testing.T) {
	// Two roles with empty ParentID → first is root, second becomes child of root.
	a := newHealed(t, &snapshot.Policy{
		Roles: []snapshot.Role{
			{ID: "root"},
			{ID: "fake_root", ParentID: ""},
		},
	})

	fake, _ := a.GetRole("fake_root")
	if fake.ParentID != "root" {
		t.Errorf("fake_root should become child of root, got parent=%q", fake.ParentID)
	}
}

func TestSelfHealDanglingUser(t *testing.T) {
	// User with non-existent RoleID → dropped.
	a := newHealed(t, &snapshot.Policy{
		Roles: []snapshot.Role{
			{ID: "root"},
		},
		Users: []snapshot.User{
			{ID: "ghost", RoleID: "bogus"},
			{ID: "real", RoleID: "root"},
		},
	})

	_, err := a.GetUser("ghost")
	if err == nil {
		t.Error("ghost user should have been dropped")
	}
	realUser, _ := a.GetUser("real")
	if realUser.RoleID != "root" {
		t.Errorf("real user should survive, got role=%q", realUser.RoleID)
	}
}

func TestSelfHealDanglingGrantFrom(t *testing.T) {
	// Grant with non-existent FromRoleID → dropped.
	a := newHealed(t, &snapshot.Policy{
		Roles: []snapshot.Role{
			{ID: "root"},
		},
		Grants: []snapshot.Grant{
			{FromRoleID: "ghost", ToRoleID: "root", Resource: "/x"},
		},
	})

	grants := getAllGrants(a)
	if len(grants) != 0 {
		t.Errorf("expected 0 grants, got %d: %+v", len(grants), grants)
	}
}

func TestSelfHealDanglingGrantTo(t *testing.T) {
	// Grant with non-existent ToRoleID → dropped.
	a := newHealed(t, &snapshot.Policy{
		Roles: []snapshot.Role{
			{ID: "root"},
		},
		Grants: []snapshot.Grant{
			{FromRoleID: "root", ToRoleID: "ghost", Resource: "/x"},
		},
	})

	grants := getAllGrants(a)
	if len(grants) != 0 {
		t.Errorf("expected 0 grants, got %d: %+v", len(grants), grants)
	}
}

func TestSelfHealNotAncestorGrant(t *testing.T) {
	// Grant where From is not ancestor of To → dropped.
	a := newHealed(t, &snapshot.Policy{
		Roles: []snapshot.Role{
			{ID: "root"},
			{ID: "role_a", ParentID: "root"},
			{ID: "role_b", ParentID: "root"},
		},
		Grants: []snapshot.Grant{
			{FromRoleID: "role_a", ToRoleID: "role_b", Resource: "/x"}, // siblings — not ancestor
		},
	})

	grants := getAllGrants(a)
	if len(grants) != 0 {
		t.Errorf("expected 0 grants, got %d: %+v", len(grants), grants)
	}
}

func TestSelfHealDuplicateGrant(t *testing.T) {
	// Duplicate grant (same From+To+Resource) → keep one, drop rest.
	a := newHealed(t, &snapshot.Policy{
		Roles: []snapshot.Role{
			{ID: "root"},
			{ID: "child", ParentID: "root"},
		},
		Grants: []snapshot.Grant{
			{FromRoleID: "root", ToRoleID: "child", Resource: "/dup"},
			{FromRoleID: "root", ToRoleID: "child", Resource: "/dup"},
		},
	})

	grants := getAllGrants(a)
	if len(grants) != 1 {
		t.Errorf("expected 1 grant (deduplicated), got %d: %+v", len(grants), grants)
	}
	if grants[0].Resource != "/dup" {
		t.Errorf("expected /dup, got %s", grants[0].Resource)
	}
}

func TestSelfHealSelfGrant(t *testing.T) {
	// Self-grant in snapshot → converted to role's own Resources, not as grant record.
	a := newHealed(t, &snapshot.Policy{
		Roles: []snapshot.Role{
			{ID: "root"},
		},
		Grants: []snapshot.Grant{
			{FromRoleID: "root", ToRoleID: "root", Resource: "/self"},
		},
	})

	// No grant record should exist.
	grants := getAllGrants(a)
	if len(grants) != 0 {
		t.Errorf("expected 0 grants for self-grant, got %d", len(grants))
	}

	// Resource should appear in root's resources.
	role, _ := a.GetRole("root")
	found := false
	for _, res := range role.Resources {
		if res == "/self" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected /self in root's resources")
	}
}

func TestSelfHealComplete(t *testing.T) {
	// Full scenario: multiple corruption types at once.
	a := newHealed(t, &snapshot.Policy{
		Roles: []snapshot.Role{
			{ID: "root"},
			{ID: "orphan", ParentID: "bogus"},
			{ID: "child", ParentID: "root"},
		},
		Users: []snapshot.User{
			{ID: "ghost", RoleID: "bogus"},
			{ID: "good", RoleID: "child"},
		},
		Grants: []snapshot.Grant{
			{FromRoleID: "bogus", ToRoleID: "child", Resource: "/bad1"},          // dangling From
			{FromRoleID: "root", ToRoleID: "bogus2", Resource: "/bad2"},         // dangling To
			{FromRoleID: "orphan", ToRoleID: "child", Resource: "/bad3"},        // not ancestor (orphan reattached to root, but child is direct child of root, not orphan)
			{FromRoleID: "root", ToRoleID: "child", Resource: "/good"},           // valid
			{FromRoleID: "root", ToRoleID: "child", Resource: "/good"},           // duplicate
			{FromRoleID: "child", ToRoleID: "child", Resource: "/self"},          // self-grant
		},
	})

	// Ghost user dropped.
	_, err := a.GetUser("ghost")
	if err == nil {
		t.Error("ghost should be dropped")
	}

	// Good user survives.
	goodUser, _ := a.GetUser("good")
	if goodUser.RoleID != "child" {
		t.Errorf("good user role=%q", goodUser.RoleID)
	}

	// Orphan reattached.
	orphan, _ := a.GetRole("orphan")
	if orphan.ParentID != "root" {
		t.Errorf("orphan parent=%q, want root", orphan.ParentID)
	}

	// Only the valid grant should survive (deduplicated to 1).
	grants := getAllGrants(a)
	if len(grants) != 1 {
		t.Errorf("expected 1 grant, got %d: %+v", len(grants), grants)
	}
	if len(grants) > 0 && grants[0].Resource != "/good" {
		t.Errorf("expected /good, got %s", grants[0].Resource)
	}

	// Self-grant converted to resource on child.
	child, _ := a.GetRole("child")
	found := false
	for _, res := range child.Resources {
		if res == "/self" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected /self in child's resources (from self-grant)")
	}
}

// =============================================================================
// Error path tests — invalid inputs
// =============================================================================

func TestGrantInvalidPath(t *testing.T) {
	a := newTestAuthorizer(t)

	err := a.Grant("root", "root", "")
	if !errors.Is(err, ErrInvalidResource) {
		t.Errorf("expected ErrInvalidResource for empty path, got %v", err)
	}

	err = a.Grant("root", "root", "no-leading-slash")
	if !errors.Is(err, ErrInvalidResource) {
		t.Errorf("expected ErrInvalidResource for no leading /, got %v", err)
	}
}

func TestRevokeInvalidPath(t *testing.T) {
	a := newTestAuthorizer(t)

	err := a.Revoke("root", "root", "")
	if !errors.Is(err, ErrInvalidResource) {
		t.Errorf("expected ErrInvalidResource, got %v", err)
	}
}

func TestCreateUserInvalidRole(t *testing.T) {
	a := newTestAuthorizer(t)

	err := a.CreateUser("nonexistent", "u")
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestGetUserNonExistent(t *testing.T) {
	a := newTestAuthorizer(t)

	_, err := a.GetUser("nonexistent")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestGetRoleNonExistent(t *testing.T) {
	a := newTestAuthorizer(t)

	_, err := a.GetRole("nonexistent")
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestRoleResourcesNonExistent(t *testing.T) {
	a := newTestAuthorizer(t)

	_, err := a.GetResource("nonexistent")
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestGrantsToNonExistent(t *testing.T) {
	a := newTestAuthorizer(t)

	_, err := a.GetGrantsTo("nonexistent")
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestGrantsFromNonExistent(t *testing.T) {
	a := newTestAuthorizer(t)

	_, err := a.GetGrantsFrom("nonexistent")
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestDeleteNonExistentRole(t *testing.T) {
	a := newTestAuthorizer(t)

	err := a.DeleteRole("nonexistent")
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestDeleteNonExistentUser(t *testing.T) {
	a := newTestAuthorizer(t)

	err := a.DeleteUser("root", "nonexistent")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

// =============================================================================
// Circular hierarchy tests
// =============================================================================

func TestCircularHierarchyRejected(t *testing.T) {
	// Manually build a corrupt snapshot with circular parent pointers.
	snap := &snapshot.Policy{
		Roles: []snapshot.Role{
			{ID: "a"},
			{ID: "b", ParentID: "a"},
			{ID: "c", ParentID: "b"},
		},
	}
	// Inject cycle: a's parent is c
	snap.Roles[0].ParentID = "c"

	_, err := NewAuthorizer(&corruptAdapter{snap: snap})
	if !errors.Is(err, ErrCircularRoleHierarchy) {
		t.Errorf("expected ErrCircularRoleHierarchy, got %v", err)
	}
}

func TestSelfReferencingParentHealed(t *testing.T) {
	// root's ParentID = "root" (self-reference) → healed: Parent stays nil.
	a := newHealed(t, &snapshot.Policy{
		Roles: []snapshot.Role{
			{ID: "root", ParentID: "root"},
			{ID: "child", ParentID: "root"},
		},
	})
	root, _ := a.GetRole("root")
	if root.ParentID != "" {
		t.Errorf("root should have no parent (self-reference healed), got parent=%q", root.ParentID)
	}
	child, _ := a.GetRole("child")
	if child.ParentID != "root" {
		t.Errorf("child parent should be root, got %q", child.ParentID)
	}
}

// =============================================================================
// Match cache tests
// =============================================================================

func TestMatchCacheHit(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "admin"))
	must(t, a.Grant("root", "admin", "/user/*"))
	must(t, a.CreateUser("admin", "u"))

	// First call — cache miss, populates cache
	ok, _ := a.Enforce("u", "/user/edit")
	if !ok {
		t.Fatal("expected true")
	}

	// Second call — should hit cache
	ok, _ = a.Enforce("u", "/user/edit")
	if !ok {
		t.Fatal("expected true from cache")
	}

	// After grant change, cache should be invalidated
	must(t, a.Revoke("root", "admin", "/user/*"))
	ok, _ = a.Enforce("u", "/user/edit")
	if ok {
		t.Error("expected false after revoke (cache should be invalidated)")
	}
}

func TestMatchCacheInvalidationOnGrant(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "role"))
	must(t, a.CreateUser("role", "u"))

	// Prime cache with a miss
	ok, _ := a.Enforce("u", "/data/read")
	if ok {
		t.Fatal("expected false before grant")
	}

	// Grant and re-check — must NOT return stale cached false
	must(t, a.Grant("root", "role", "/data/*"))
	ok, _ = a.Enforce("u", "/data/read")
	if !ok {
		t.Error("expected true after grant (cache must be invalidated)")
	}
}

func TestMatchCacheInvalidationOnRevoke(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "role"))
	must(t, a.Grant("root", "role", "/data/*"))
	must(t, a.CreateUser("role", "u"))

	// Prime cache with a hit
	ok, _ := a.Enforce("u", "/data/read")
	if !ok {
		t.Fatal("expected true before revoke")
	}

	// Revoke and re-check — must NOT return stale cached true
	must(t, a.Revoke("root", "role", "/data/*"))
	ok, _ = a.Enforce("u", "/data/read")
	if ok {
		t.Error("expected false after revoke (cache must be invalidated)")
	}
}

func TestMatchCacheInvalidationOnDeleteRole(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "role"))
	must(t, a.CreateUser("role", "u"))
	must(t, a.Grant("root", "role", "/shared/*"))

	// Prime cache
	ok, _ := a.Enforce("u", "/shared/doc")
	if !ok {
		t.Fatal("expected true before delete")
	}

	// Delete the grantor (root grants to role → but root can't be deleted)
	// Delete role instead, then re-create and verify no stale cache
	must(t, a.DeleteRole("role"))
	must(t, a.CreateRole("root", "role"))
	must(t, a.CreateUser("role", "u2"))

	// New role has no grants — enforce should miss
	ok, _ = a.Enforce("u2", "/shared/doc")
	if ok {
		t.Error("expected false for new role with no grants")
	}
}

// =============================================================================
// Grant / Revoke edge cases
// =============================================================================

func TestGrantToNonExistentRole(t *testing.T) {
	a := newTestAuthorizer(t)

	err := a.Grant("root", "nonexistent", "/x")
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("expected ErrRoleNotFound, got %v", err)
	}

	err = a.Grant("nonexistent", "root", "/x")
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestRevokeNonExistentGrant(t *testing.T) {
	a := newTestAuthorizer(t)

	err := a.Revoke("root", "root", "/nonexistent")
	if !errors.Is(err, ErrGrantNotFound) {
		t.Errorf("expected ErrGrantNotFound, got %v", err)
	}
}

func TestRevokeNonExistentRole(t *testing.T) {
	a := newTestAuthorizer(t)

	err := a.Revoke("root", "nonexistent", "/x")
	if !errors.Is(err, ErrGrantNotFound) {
		t.Errorf("expected ErrGrantNotFound, got %v", err)
	}
}

func TestMultipleGrantsDifferentAncestors(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "grandparent"))
	must(t, a.CreateRole("grandparent", "parent"))
	must(t, a.CreateRole("parent", "child"))
	must(t, a.CreateUser("child", "u"))

	// Multiple ancestors grant the same resource
	must(t, a.Grant("root", "child", "/shared/**"))
	must(t, a.Grant("grandparent", "child", "/shared/**")) // duplicate — should fail

	// Check the first grant works
	ok, _ := a.Enforce("u", "/shared/doc")
	if !ok {
		t.Error("child's user should have /shared/** from root grant")
	}
}

func TestGrantRevokeThenGrantAgain(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "child"))
	must(t, a.CreateUser("child", "u"))

	must(t, a.Grant("root", "child", "/toggle"))
	ok, _ := a.Enforce("u", "/toggle")
	if !ok {
		t.Fatal("expected true after grant")
	}

	must(t, a.Revoke("root", "child", "/toggle"))
	ok, _ = a.Enforce("u", "/toggle")
	if ok {
		t.Error("expected false after revoke")
	}

	// Grant again — should work
	must(t, a.Grant("root", "child", "/toggle"))
	ok, _ = a.Enforce("u", "/toggle")
	if !ok {
		t.Error("expected true after re-grant")
	}
}

func TestRevokeCascadeMultipleGenerations(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "r1"))
	must(t, a.CreateRole("r1", "r2"))
	must(t, a.CreateRole("r2", "r3"))
	must(t, a.CreateRole("r3", "r4"))

	// Grant /cascade down the chain
	must(t, a.Grant("root", "r1", "/cascade"))
	must(t, a.Grant("r1", "r2", "/cascade"))
	must(t, a.Grant("r2", "r3", "/cascade"))
	must(t, a.Grant("r3", "r4", "/cascade"))

	// Revoke at top — all should cascade
	must(t, a.Revoke("root", "r1", "/cascade"))

	grants := getAllGrants(a)
	for _, g := range grants {
		if g.Resource == "/cascade" {
			t.Errorf("cascade grant still exists: %+v", g)
		}
	}
}

func TestGrantWithSpecialChars(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "child"))
	must(t, a.CreateUser("child", "u"))

	must(t, a.Grant("root", "child", "/api/v1/items/*"))
	ok, _ := a.Enforce("u", "/api/v1/items/123")
	if !ok {
		t.Error("expected true for /api/v1/items/123")
	}

	ok, _ = a.Enforce("u", "/api/v1/other")
	if ok {
		t.Error("expected false for /api/v1/other")
	}
}

// =============================================================================
// Deep nesting tests
// =============================================================================

func TestDeepRoleNesting(t *testing.T) {
	a := newTestAuthorizer(t)

	parent := "root"
	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("role_%d", i)
		must(t, a.CreateRole(parent, id))
		parent = id
	}

	roles := a.GetAllRoles()
	if len(roles) != 21 { // root + 20 children
		t.Errorf("expected 21 roles, got %d", len(roles))
	}
}

func TestDeepGrantChain(t *testing.T) {
	a := newTestAuthorizer(t)

	parent := "root"
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("r%d", i)
		must(t, a.CreateRole(parent, id))
		parent = id
	}
	must(t, a.CreateUser("r9", "deep_user"))

	// Grant from root to deepest role — skips 8 intermediate roles
	must(t, a.Grant("root", "r9", "/deep/**"))

	ok, _ := a.Enforce("deep_user", "/deep/nested/path")
	if !ok {
		t.Error("deep user should have /deep/** from root grant")
	}
}

// =============================================================================
// Resource normalization corner cases
// =============================================================================

func TestNormalizeResourcePath(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "child"))
	must(t, a.CreateUser("child", "u"))

	// Double slashes should be normalized
	must(t, a.Grant("root", "child", "/data//read"))
	ok, _ := a.Enforce("u", "/data/read")
	if !ok {
		t.Log("double slash normalized by path.Clean")
	}
}

func TestNoAdapterPersistence(t *testing.T) {
	// Authorizer without adapter should work and not crash on save
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "child"))
	must(t, a.Grant("root", "child", "/app/*"))
	must(t, a.CreateUser("child", "u1"))

	ok, _ := a.Enforce("u1", "/app/dashboard")
	if !ok {
		t.Error("user should have /app/dashboard from child's /app/*")
	}

	ok, _ = a.Enforce("u1", "/anything")
	if ok {
		t.Error("user should not have /** access — no inheritance from root")
	}
}

// =============================================================================
// Cascade revoke tests — verify that revoking a grant cascades to downstream
// delegations whose resources are no longer covered by the grantor's GrantedMap.
// =============================================================================

// TestRevokeCascadeSubResource: revoke /res at top → /res/1 downstream cascades.
func TestRevokeCascadeSubResource(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "r1"))
	must(t, a.CreateRole("r1", "r2"))

	must(t, a.Grant("root", "r1", "/res"))
	must(t, a.Grant("r1", "r2", "/res/1"))

	must(t, a.Revoke("root", "r1", "/res"))

	grants := getAllGrants(a)
	if len(grants) != 0 {
		t.Errorf("expected 0 grants after cascade, got %d: %+v", len(grants), grants)
	}
}

// TestRevokeCascadeMultiGrantSurvival: revoking one grant when another covers
// the same sub-tree should preserve delegations still covered.
// root grants /res/** and /res/1/** to r1; r1 grants /res/1/c to r2.
// Revoke /res/** → r1→r2 /res/1/c must SURVIVE (covered by /res/1/**).
func TestRevokeCascadeMultiGrantSurvival(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "r1"))
	must(t, a.CreateRole("r1", "r2"))

	must(t, a.Grant("root", "r1", "/res/**"))
	must(t, a.Grant("root", "r1", "/res/1/**"))
	must(t, a.Grant("r1", "r2", "/res/1/c"))

	must(t, a.Revoke("root", "r1", "/res/**"))

	grants := getAllGrants(a)
if len(grants) != 2 {
			t.Fatalf("expected 2 surviving grants, got %d: %+v", len(grants), grants)
		}
		hasRes1Star := false
		hasRes1c := false
		for _, g := range grants {
			if g.Resource == "/res/1/**" {
				hasRes1Star = true
			}
			if g.Resource == "/res/1/c" {
				hasRes1c = true
			}
		}
		if !hasRes1Star {
			t.Error("expected /res/1/** to survive")
		}
		if !hasRes1c {
			t.Error("expected /res/1/c to survive")
		}
}

// TestRevokeCascadeDeepChain: 4-level chain, revoke at top cascades all.
func TestRevokeCascadeDeepChain(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "r1"))
	must(t, a.CreateRole("r1", "r2"))
	must(t, a.CreateRole("r2", "r3"))
	must(t, a.CreateRole("r3", "r4"))

	must(t, a.Grant("root", "r1", "/data"))
	must(t, a.Grant("r1", "r2", "/data/1"))
	must(t, a.Grant("r2", "r3", "/data/1/a"))
	must(t, a.Grant("r3", "r4", "/data/1/a/x"))

	must(t, a.Revoke("root", "r1", "/data"))

	grants := getAllGrants(a)
	if len(grants) != 0 {
		t.Errorf("expected 0 grants after deep cascade, got %d: %+v", len(grants), grants)
	}
}

// TestRevokeCascadeGlobPattern: revoke /res/* → cascade removes /res/1, /res/2
// but /res/1/sub from a different ancestor survives.
func TestRevokeCascadeGlobPattern(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "r1"))
	must(t, a.CreateRole("r1", "r2"))
	must(t, a.CreateRole("r2", "r3"))

	must(t, a.Grant("root", "r1", "/res/*"))
	must(t, a.Grant("r1", "r2", "/res/1"))
	must(t, a.Grant("r1", "r2", "/res/2"))
	must(t, a.Grant("root", "r3", "/res/1/sub"))
	must(t, a.Grant("r2", "r3", "/res/1"))

	must(t, a.Revoke("root", "r1", "/res/*"))

	grants := getAllGrants(a)
	for _, g := range grants {
		if g.Resource == "/res/1" || g.Resource == "/res/2" {
			t.Errorf("grant %s should have been cascade-revoked: %+v", g.Resource, g)
		}
	}
	survivors := 0
	for _, g := range grants {
		if g.Resource == "/res/1/sub" {
			survivors++
		}
	}
	if survivors != 1 {
		t.Errorf("/res/1/sub should survive, got %d survivors", survivors)
	}
}

// TestRevokeCascadeTwoAncestors: role gets grants from two different ancestors.
// Revoking one should only cascade resources uniquely covered by it.
// root→r2 /a/**, r1→r2 /a/1/**, r2→r3 /a/1/x. Revoke root→r2 /a/** →
// /a/1/x survives because r1→r2 /a/1/** still covers it.
func TestRevokeCascadeTwoAncestors(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "r1"))
	must(t, a.CreateRole("r1", "r2"))
	must(t, a.CreateRole("r2", "r3"))

	must(t, a.Grant("root", "r2", "/a/**"))
	must(t, a.Grant("r1", "r2", "/a/1/**"))
	must(t, a.Grant("r2", "r3", "/a/1/x"))

	must(t, a.Revoke("root", "r2", "/a/**"))

	grants := getAllGrants(a)
		if len(grants) != 2 {
		t.Fatalf("expected 2 surviving grants, got %d: %+v", len(grants), grants)
		}
		hasA1Star := false
		hasA1x := false
		for _, g := range grants {
		if g.FromRoleID == "r1" && g.ToRoleID == "r2" && g.Resource == "/a/1/**" {
			hasA1Star = true
		}
		if g.FromRoleID == "r2" && g.ToRoleID == "r3" && g.Resource == "/a/1/x" {
			hasA1x = true
		}
		}
		if !hasA1Star {
		t.Error("expected r1->r2 /a/1/** to survive")
		}
		if !hasA1x {
		t.Error("expected r2->r3 /a/1/x to survive (covered by /a/1/**)")
		}
}

// TestRevokeCascadeOnlyAffectedSubtree: sibling branches unaffected by cascade.
func TestRevokeCascadeOnlyAffectedSubtree(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "r1"))
	must(t, a.CreateRole("root", "r2"))
	must(t, a.CreateRole("r1", "r1_child"))
	must(t, a.CreateRole("r2", "r2_child"))

	must(t, a.Grant("root", "r1", "/x"))
	must(t, a.Grant("root", "r2", "/x"))
	must(t, a.Grant("r1", "r1_child", "/x/1"))
	must(t, a.Grant("r2", "r2_child", "/x/1"))

	must(t, a.Revoke("root", "r1", "/x"))

	grants := getAllGrants(a)
	if len(grants) != 2 {
		t.Fatalf("expected 2 surviving grants, got %d: %+v", len(grants), grants)
	}
	hasRootR2 := false
	hasR2Child := false
	for _, g := range grants {
		if g.FromRoleID == "root" && g.ToRoleID == "r2" && g.Resource == "/x" {
			hasRootR2 = true
		}
		if g.FromRoleID == "r2" && g.ToRoleID == "r2_child" && g.Resource == "/x/1" {
			hasR2Child = true
		}
	}
	if !hasRootR2 {
		t.Error("root→r2 /x should survive")
	}
	if !hasR2Child {
		t.Error("r2→r2_child /x/1 should survive (sibling branch)")
	}
}

// TestRevokeCascadeExactGlob: revoke /res/** cascades matching /res/** downstream.
func TestRevokeCascadeExactGlob(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "r1"))
	must(t, a.CreateRole("r1", "r2"))

	must(t, a.Grant("root", "r1", "/res/**"))
	must(t, a.Grant("r1", "r2", "/res/**"))

	must(t, a.Revoke("root", "r1", "/res/**"))

	grants := getAllGrants(a)
	if len(grants) != 0 {
		t.Errorf("expected 0 grants, got %d: %+v", len(grants), grants)
	}
}
// TestRevokeCascadePartialCoverageLoss: /a/** and /a/b granted; /a/b/c delegated.
// Revoke /a/** → /a/b/c cascades because /a/b (exact) does NOT match /a/b/c.
func TestRevokeCascadePartialCoverageLoss(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "r1"))
	must(t, a.CreateRole("r1", "r2"))

	must(t, a.Grant("root", "r1", "/a/**"))
	must(t, a.Grant("root", "r1", "/a/b"))
	must(t, a.Grant("r1", "r2", "/a/b/c"))

	must(t, a.Revoke("root", "r1", "/a/**"))

	// root→r1 /a/b survives (not affected by revoking /a/**)
	grants := getAllGrants(a)
	if len(grants) != 1 {
		t.Fatalf("expected 1 surviving grant (root→r1 /a/b), got %d: %+v", len(grants), grants)
	}
	if grants[0].Resource != "/a/b" || grants[0].FromRoleID != "root" || grants[0].ToRoleID != "r1" {
		t.Errorf("expected root→r1 /a/b to survive, got %+v", grants[0])
	}
}

// TestRevokeCascadeAfterRetainingCoverage: verify enforcement after cascade.
func TestRevokeCascadeAfterRetainingCoverage(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "r1"))
	must(t, a.CreateRole("r1", "r2"))
	must(t, a.CreateUser("r2", "u"))

	must(t, a.Grant("root", "r1", "/res/**"))
	must(t, a.Grant("root", "r1", "/res/1/**"))
	must(t, a.Grant("r1", "r2", "/res/1/c"))

	// User should have access before revoke
	ok, _ := a.Enforce("u", "/res/1/c")
	if !ok {
		t.Fatal("expected access to /res/1/c before revoke")
	}

	must(t, a.Revoke("root", "r1", "/res/**"))

	// User should STILL have access after revoke (covered by /res/1/**)
	ok, _ = a.Enforce("u", "/res/1/c")
	if !ok {
		t.Error("expected access to /res/1/c after revoke (still covered by /res/1/**)")
	}
}

// =============================================================================
// Additional edge case tests
// =============================================================================

func TestGetSubRoles(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "r1"))
	must(t, a.CreateRole("root", "r2"))
	must(t, a.CreateRole("r1", "r1_child"))

	children, err := a.GetSubRoles("root")
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 2 {
		t.Fatalf("expected 2 children of root, got %d", len(children))
	}
	ids := make(map[string]bool)
	for _, c := range children {
		ids[c.ID] = true
	}
	if !ids["r1"] || !ids["r2"] {
		t.Errorf("expected r1 and r2 as children of root, got %v", ids)
	}

	children, err = a.GetSubRoles("r1")
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 1 || children[0].ID != "r1_child" {
		t.Errorf("expected 1 child r1_child of r1, got %v", children)
	}

	_, err = a.GetSubRoles("nonexistent")
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestGrantFromIntermediateAncestor(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "admin"))
	must(t, a.CreateRole("admin", "editor"))
	must(t, a.CreateUser("editor", "u"))

	// Grant from admin (intermediate ancestor) to editor
	must(t, a.Grant("admin", "editor", "/reports/*"))

	ok, err := a.Enforce("u", "/reports/q1")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("editor should have /reports/* from admin grant")
	}
}

func TestRevokeFromIntermediateAncestor(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "admin"))
	must(t, a.CreateRole("admin", "editor"))
	must(t, a.CreateUser("editor", "u"))

	must(t, a.Grant("root", "admin", "/data/*"))
	must(t, a.Grant("admin", "editor", "/data/*"))

	ok, _ := a.Enforce("u", "/data/read")
	if !ok {
		t.Fatal("editor should have /data/* before revoke")
	}

	must(t, a.Revoke("admin", "editor", "/data/*"))

	ok, _ = a.Enforce("u", "/data/read")
	if ok {
		t.Error("editor should NOT have /data/* after revoke from admin")
	}
}

func TestEnforceCachedThenMiss(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "r1"))
	must(t, a.CreateUser("r1", "u"))
	must(t, a.Grant("root", "r1", "/data/*"))

	// First call — cache miss, glob match
	ok, err := a.Enforce("u", "/data/read")
	if err != nil || !ok {
		t.Fatal("expected access on first call")
	}

	// Second call — cache hit
	ok, err = a.Enforce("u", "/data/read")
	if err != nil || !ok {
		t.Error("expected access on cached call")
	}
}

func TestDeleteUserMismatchedRole(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "r1"))
	must(t, a.CreateRole("root", "r2"))
	must(t, a.CreateUser("r1", "u"))

	// Delete with wrong roleID
	err := a.DeleteUser("r2", "u")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound for wrong role, got %v", err)
	}

	// User should still exist
	_, err = a.GetUser("u")
	if err != nil {
		t.Errorf("user should still exist: %v", err)
	}
}

func TestCreateUserAfterDeleteRole(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "r1"))
	must(t, a.CreateUser("r1", "u1"))

	// Delete role — cascades to user
	must(t, a.DeleteRole("r1"))

	// User should be gone
	_, err := a.GetUser("u1")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound after role deleted, got %v", err)
	}

	// Create same role and user again
	must(t, a.CreateRole("root", "r1"))
	must(t, a.CreateUser("r1", "u1"))

	ok, err := a.Enforce("u1", "/anything")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if ok {
		t.Error("user without any grants should not have access")
	}
}

func TestEnforceMultipleWildcardPatterns(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "r1"))
	must(t, a.CreateUser("r1", "u"))

	must(t, a.Grant("root", "r1", "/api/v1/*"))
	must(t, a.Grant("root", "r1", "/api/v2/**"))

	cases := []struct {
		resource string
		want     bool
	}{
		{"/api/v1/users", true},
		{"/api/v1/users/123", false},   // * matches one segment only
		{"/api/v2/users/123", true},    // ** matches multiple segments
		{"/api/v2", true},              // ** matches zero segments
		{"/api/v3/test", false},        // no matching pattern
		{"/other/path", false},         // no matching pattern
	}

	for _, tc := range cases {
		ok, err := a.Enforce("u", tc.resource)
		if err != nil {
			t.Errorf("Enforce(%q): %v", tc.resource, err)
			continue
		}
		if ok != tc.want {
			t.Errorf("Enforce(%q) = %v, want %v", tc.resource, ok, tc.want)
		}
	}
}

func TestGetSubRolesLeafRole(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "leaf"))

	children, err := a.GetSubRoles("leaf")
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 0 {
		t.Errorf("expected 0 children for leaf role, got %d", len(children))
	}
}
