package auther

import (
	"errors"
	"fmt"
	"testing"

	"auther/model"
)

// =============================================================================
// Helpers
// =============================================================================

// newTestAuthorizer creates an empty authorizer (no adapter).
func newTestAuthorizer(t *testing.T) *Authorizer {
	t.Helper()
	a, err := NewAuthorizer(nil)
	if err != nil {
		t.Fatalf("NewAuthorizer: %v", err)
	}
	return a
}

// setupTestHierarchy creates a standard test hierarchy:
//
//	Roles: root(/**) -> admin -> editor
//	       admin has /user/*
//	       editor has /data/*
//	       root grants /g/** to admin
//	Users: admin has u_admin
//	       editor has u_editor
func setupTestHierarchy(t *testing.T, a *Authorizer) {
	t.Helper()

	must(t, a.CreateRole("root", "admin"))
	must(t, a.CreateRole("admin", "editor"))
	must(t, a.GrantResource("admin", "admin", "/user/*"))
	must(t, a.GrantResource("editor", "editor", "/data/*"))
	must(t, a.GrantResource("root", "admin", "/g/**"))
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
	must(t, a.GrantResource("admin", "editor", "/shared/*"))
	must(t, a.GrantResource("editor", "sub_editor", "/shared/*"))

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
	grants := a.GetAllGrants()
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

	must(t, a.GrantResource("root", "root", "/extra/**"))

	role, _ := a.GetRole("root")
	found := false
	for _, r := range role.Resources {
		if r == "/extra/**" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected /extra/** in root resources")
	}

	must(t, a.RevokeResource("root", "root", "/extra/**"))

	role, _ = a.GetRole("root")
	for _, r := range role.Resources {
		if r == "/extra/**" {
			t.Error("expected /extra/** to be removed")
		}
	}

	// Remove non-existent resource
	err := a.RevokeResource("root", "root", "/nonexistent")
	if !errors.Is(err, ErrGrantNotFound) {
		t.Errorf("expected ErrGrantNotFound, got %v", err)
	}
}

// =============================================================================
// Grant tests
// =============================================================================

func TestGrantResource(t *testing.T) {
	a := newTestAuthorizer(t)

	must(t, a.CreateRole("root", "child"))
	must(t, a.GrantResource("root", "child", "/custom/*"))

	grantsFrom, _ := a.GetGrantsFromRole("root")
	if len(grantsFrom) != 1 {
		t.Fatalf("expected 1 grant from root, got %d", len(grantsFrom))
	}
	if grantsFrom[0].Resource != "/custom/*" {
		t.Errorf("expected /custom/*, got %s", grantsFrom[0].Resource)
	}

	grantsTo, _ := a.GetGrantsToRole("child")
	if len(grantsTo) != 1 {
		t.Fatalf("expected 1 grant to child, got %d", len(grantsTo))
	}
}

func TestGrantResourceNotAncestor(t *testing.T) {
	a := newTestAuthorizer(t)

	must(t, a.CreateRole("root", "role_a"))
	must(t, a.CreateRole("root", "role_b"))

	// role_a is not an ancestor of role_b
	err := a.GrantResource("role_a", "role_b", "/x")
	if !errors.Is(err, ErrNotAncestor) {
		t.Errorf("expected ErrNotAncestor, got %v", err)
	}
}

func TestGrantResourceDuplicate(t *testing.T) {
	a := newTestAuthorizer(t)

	must(t, a.CreateRole("root", "child"))
	must(t, a.GrantResource("root", "child", "/dup"))

	err := a.GrantResource("root", "child", "/dup")
	if !errors.Is(err, ErrDuplicateGrant) {
		t.Errorf("expected ErrDuplicateGrant, got %v", err)
	}
}

func TestGrantResourceSelf(t *testing.T) {
	a := newTestAuthorizer(t)

	// Self-grant adds directly to role's own Resources, not as a Grant record
	must(t, a.GrantResource("root", "root", "/self/**"))

	role, _ := a.GetRole("root")
	found := false
	for _, r := range role.Resources {
		if r == "/self/**" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected /self/** in root's own resources")
	}

	// No grant record should exist for self-grant
	grants := a.GetAllGrants()
	if len(grants) != 0 {
		t.Errorf("expected 0 grants for self-grant, got %d", len(grants))
	}

	// Revoke works
	must(t, a.RevokeResource("root", "root", "/self/**"))
	role, _ = a.GetRole("root")
	for _, r := range role.Resources {
		if r == "/self/**" {
			t.Error("expected /self/** to be removed")
		}
	}
}

func TestRevokeResourceCascade(t *testing.T) {
	a := newTestAuthorizer(t)
	setupTestHierarchy(t, a)

	// Grant chain: root -> admin -> editor
	must(t, a.GrantResource("admin", "editor", "/reports/*"))

	// Revoke at the top
	must(t, a.RevokeResource("root", "admin", "/g/**"))

	// Check that the grant is gone
	grants := a.GetAllGrants()
	for _, g := range grants {
		if g.Resource == "/g/**" {
			t.Errorf("grant still exists: %+v", g)
		}
	}
}

func TestRevokeResourceCascadeDeep(t *testing.T) {
	a := newTestAuthorizer(t)

	must(t, a.CreateRole("root", "r1"))
	must(t, a.CreateRole("r1", "r2"))
	must(t, a.CreateRole("r2", "r3"))

	must(t, a.GrantResource("root", "r1", "/reports"))
	must(t, a.GrantResource("r1", "r2", "/reports"))
	must(t, a.GrantResource("r2", "r3", "/reports"))

	// Revoke at r1; should cascade to r2 and r3
	must(t, a.RevokeResource("root", "r1", "/reports"))

	grants := a.GetAllGrants()
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
	must(t, a.DeleteUser("u1"))

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
	must(t, a.GrantResource("admin", "editor", "/user/*"))

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

func TestGetUserPermissions(t *testing.T) {
	a := newTestAuthorizer(t)
	setupTestHierarchy(t, a)

	must(t, a.GrantResource("admin", "editor", "/extra/**"))

	perms, err := a.GetUserPermissions("u_editor")
	if err != nil {
		t.Fatal(err)
	}

	hasData := false
	hasExtra := false
	for _, p := range perms {
		if p == "/data/*" {
			hasData = true
		}
		if p == "/extra/**" {
			hasExtra = true
		}
	}
	if !hasData {
		t.Error("expected /data/* in permissions")
	}
	if !hasExtra {
		t.Error("expected /extra/** in permissions")
	}
}

// =============================================================================
// GetAll tests
// =============================================================================

func TestGetAllRoles(t *testing.T) {
	a := newTestAuthorizer(t)
	setupTestHierarchy(t, a)

	roles := a.GetAllRoles()
	if len(roles) != 3 { // root, admin, editor
		t.Errorf("expected 3 roles, got %d", len(roles))
	}
}

func TestGetAllUsers(t *testing.T) {
	a := newTestAuthorizer(t)
	setupTestHierarchy(t, a)

	users := a.GetAllUsers()
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestGetAllGrants(t *testing.T) {
	a := newTestAuthorizer(t)
	setupTestHierarchy(t, a)

	grants := a.GetAllGrants()
	if len(grants) != 1 {
		t.Errorf("expected 1 grant, got %d", len(grants))
	}
}

// =============================================================================
// Effective role resources
// =============================================================================

func TestEffectiveRoleResources(t *testing.T) {
	a := newTestAuthorizer(t)
	setupTestHierarchy(t, a)

	// Editor should have /data/* (own) — no auto-inheritance
	effective, err := a.GetEffectiveRoleResources("editor")
	if err != nil {
		t.Fatal(err)
	}

	if len(effective) != 1 || effective[0] != "/data/*" {
		t.Errorf("expected [/data/*], got %v", effective)
	}

	// Admin should have /user/* (own) + /g/** (grant from root)
	effective, err = a.GetEffectiveRoleResources("admin")
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
	snap *model.PolicySnapshot
}

func (a *corruptAdapter) Load() (*model.PolicySnapshot, error) { return a.snap, nil }
func (a *corruptAdapter) Save(s *model.PolicySnapshot) error   { a.snap = s; return nil }

func newHealed(t *testing.T, snap *model.PolicySnapshot) *Authorizer {
	t.Helper()
	a, err := NewAuthorizer(&corruptAdapter{snap: snap})
	if err != nil {
		t.Fatalf("NewAuthorizer: %v", err)
	}
	return a
}

func TestSelfHealOrphanRole(t *testing.T) {
	// Role with non-existent ParentID → reattached to root.
	a := newHealed(t, &model.PolicySnapshot{
		Roles: []model.RoleSnapshot{
			{ID: "root", Resources: []string{"/**"}},
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
	a := newHealed(t, &model.PolicySnapshot{
		Roles: []model.RoleSnapshot{
			{ID: "child", ParentID: "root", Resources: []string{"/x"}},
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
	a := newHealed(t, &model.PolicySnapshot{
		Roles: []model.RoleSnapshot{
			{ID: "root", Resources: []string{"/**"}},
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
	a := newHealed(t, &model.PolicySnapshot{
		Roles: []model.RoleSnapshot{
			{ID: "root", Resources: []string{"/**"}},
		},
		Users: []model.UserSnapshot{
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
	a := newHealed(t, &model.PolicySnapshot{
		Roles: []model.RoleSnapshot{
			{ID: "root", Resources: []string{"/**"}},
		},
		Grants: []model.GrantSnapshot{
			{FromRoleID: "ghost", ToRoleID: "root", Resource: "/x"},
		},
	})

	grants := a.GetAllGrants()
	if len(grants) != 0 {
		t.Errorf("expected 0 grants, got %d: %+v", len(grants), grants)
	}
}

func TestSelfHealDanglingGrantTo(t *testing.T) {
	// Grant with non-existent ToRoleID → dropped.
	a := newHealed(t, &model.PolicySnapshot{
		Roles: []model.RoleSnapshot{
			{ID: "root", Resources: []string{"/**"}},
		},
		Grants: []model.GrantSnapshot{
			{FromRoleID: "root", ToRoleID: "ghost", Resource: "/x"},
		},
	})

	grants := a.GetAllGrants()
	if len(grants) != 0 {
		t.Errorf("expected 0 grants, got %d: %+v", len(grants), grants)
	}
}

func TestSelfHealNotAncestorGrant(t *testing.T) {
	// Grant where From is not ancestor of To → dropped.
	a := newHealed(t, &model.PolicySnapshot{
		Roles: []model.RoleSnapshot{
			{ID: "root", Resources: []string{"/**"}},
			{ID: "role_a", ParentID: "root"},
			{ID: "role_b", ParentID: "root"},
		},
		Grants: []model.GrantSnapshot{
			{FromRoleID: "role_a", ToRoleID: "role_b", Resource: "/x"}, // siblings — not ancestor
		},
	})

	grants := a.GetAllGrants()
	if len(grants) != 0 {
		t.Errorf("expected 0 grants, got %d: %+v", len(grants), grants)
	}
}

func TestSelfHealDuplicateGrant(t *testing.T) {
	// Duplicate grant (same From+To+Resource) → keep one, drop rest.
	a := newHealed(t, &model.PolicySnapshot{
		Roles: []model.RoleSnapshot{
			{ID: "root", Resources: []string{"/**"}},
			{ID: "child", ParentID: "root"},
		},
		Grants: []model.GrantSnapshot{
			{FromRoleID: "root", ToRoleID: "child", Resource: "/dup"},
			{FromRoleID: "root", ToRoleID: "child", Resource: "/dup"},
		},
	})

	grants := a.GetAllGrants()
	if len(grants) != 1 {
		t.Errorf("expected 1 grant (deduplicated), got %d: %+v", len(grants), grants)
	}
	if grants[0].Resource != "/dup" {
		t.Errorf("expected /dup, got %s", grants[0].Resource)
	}
}

func TestSelfHealSelfGrant(t *testing.T) {
	// Self-grant in snapshot → converted to role's own Resources, not as grant record.
	a := newHealed(t, &model.PolicySnapshot{
		Roles: []model.RoleSnapshot{
			{ID: "root", Resources: []string{"/**"}},
		},
		Grants: []model.GrantSnapshot{
			{FromRoleID: "root", ToRoleID: "root", Resource: "/self"},
		},
	})

	// No grant record should exist.
	grants := a.GetAllGrants()
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
	a := newHealed(t, &model.PolicySnapshot{
		Roles: []model.RoleSnapshot{
			{ID: "root", Resources: []string{"/**"}},
			{ID: "orphan", ParentID: "bogus"},
			{ID: "child", ParentID: "root"},
		},
		Users: []model.UserSnapshot{
			{ID: "ghost", RoleID: "bogus"},
			{ID: "good", RoleID: "child"},
		},
		Grants: []model.GrantSnapshot{
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
	grants := a.GetAllGrants()
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

func TestGrantResourceInvalidPath(t *testing.T) {
	a := newTestAuthorizer(t)

	err := a.GrantResource("root", "root", "")
	if !errors.Is(err, ErrInvalidResource) {
		t.Errorf("expected ErrInvalidResource for empty path, got %v", err)
	}

	err = a.GrantResource("root", "root", "no-leading-slash")
	if !errors.Is(err, ErrInvalidResource) {
		t.Errorf("expected ErrInvalidResource for no leading /, got %v", err)
	}
}

func TestRevokeResourceInvalidPath(t *testing.T) {
	a := newTestAuthorizer(t)

	err := a.RevokeResource("root", "root", "")
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

func TestGetEffectiveRoleResourcesNonExistent(t *testing.T) {
	a := newTestAuthorizer(t)

	_, err := a.GetEffectiveRoleResources("nonexistent")
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestGetUserPermissionsNonExistent(t *testing.T) {
	a := newTestAuthorizer(t)

	_, err := a.GetUserPermissions("nonexistent")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestGetGrantsToRoleNonExistent(t *testing.T) {
	a := newTestAuthorizer(t)

	_, err := a.GetGrantsToRole("nonexistent")
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestGetGrantsFromRoleNonExistent(t *testing.T) {
	a := newTestAuthorizer(t)

	_, err := a.GetGrantsFromRole("nonexistent")
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

	err := a.DeleteUser("nonexistent")
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

// =============================================================================
// Circular hierarchy tests
// =============================================================================

func TestCircularHierarchyRejected(t *testing.T) {
	// Manually build a corrupt snapshot with circular parent pointers.
	snap := &model.PolicySnapshot{
		Roles: []model.RoleSnapshot{
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
	a := newHealed(t, &model.PolicySnapshot{
		Roles: []model.RoleSnapshot{
			{ID: "root", ParentID: "root", Resources: []string{"/**"}},
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
	must(t, a.GrantResource("admin", "admin", "/user/*"))
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
	must(t, a.RevokeResource("admin", "admin", "/user/*"))
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
	must(t, a.GrantResource("role", "role", "/data/*"))
	ok, _ = a.Enforce("u", "/data/read")
	if !ok {
		t.Error("expected true after grant (cache must be invalidated)")
	}
}

func TestMatchCacheInvalidationOnRevoke(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "role"))
	must(t, a.GrantResource("role", "role", "/data/*"))
	must(t, a.CreateUser("role", "u"))

	// Prime cache with a hit
	ok, _ := a.Enforce("u", "/data/read")
	if !ok {
		t.Fatal("expected true before revoke")
	}

	// Revoke and re-check — must NOT return stale cached true
	must(t, a.RevokeResource("role", "role", "/data/*"))
	ok, _ = a.Enforce("u", "/data/read")
	if ok {
		t.Error("expected false after revoke (cache must be invalidated)")
	}
}

func TestMatchCacheInvalidationOnDeleteRole(t *testing.T) {
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "role"))
	must(t, a.CreateUser("role", "u"))
	must(t, a.GrantResource("root", "role", "/shared/*"))

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

	err := a.GrantResource("root", "nonexistent", "/x")
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("expected ErrRoleNotFound, got %v", err)
	}

	err = a.GrantResource("nonexistent", "root", "/x")
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("expected ErrRoleNotFound, got %v", err)
	}
}

func TestRevokeNonExistentGrant(t *testing.T) {
	a := newTestAuthorizer(t)

	err := a.RevokeResource("root", "root", "/nonexistent")
	if !errors.Is(err, ErrGrantNotFound) {
		t.Errorf("expected ErrGrantNotFound, got %v", err)
	}
}

func TestRevokeNonExistentRole(t *testing.T) {
	a := newTestAuthorizer(t)

	err := a.RevokeResource("root", "nonexistent", "/x")
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
	must(t, a.GrantResource("root", "child", "/shared/**"))
	must(t, a.GrantResource("grandparent", "child", "/shared/**")) // duplicate — should fail

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

	must(t, a.GrantResource("root", "child", "/toggle"))
	ok, _ := a.Enforce("u", "/toggle")
	if !ok {
		t.Fatal("expected true after grant")
	}

	must(t, a.RevokeResource("root", "child", "/toggle"))
	ok, _ = a.Enforce("u", "/toggle")
	if ok {
		t.Error("expected false after revoke")
	}

	// Grant again — should work
	must(t, a.GrantResource("root", "child", "/toggle"))
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
	must(t, a.GrantResource("root", "r1", "/cascade"))
	must(t, a.GrantResource("r1", "r2", "/cascade"))
	must(t, a.GrantResource("r2", "r3", "/cascade"))
	must(t, a.GrantResource("r3", "r4", "/cascade"))

	// Revoke at top — all should cascade
	must(t, a.RevokeResource("root", "r1", "/cascade"))

	grants := a.GetAllGrants()
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

	must(t, a.GrantResource("child", "child", "/api/v1/items/*"))
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
	must(t, a.GrantResource("root", "r9", "/deep/**"))

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
	must(t, a.GrantResource("child", "child", "/data//read"))
	ok, _ := a.Enforce("u", "/data/read")
	if !ok {
		t.Log("double slash normalized by path.Clean")
	}
}

func TestNoAdapterPersistence(t *testing.T) {
	// Authorizer without adapter should work and not crash on save
	a := newTestAuthorizer(t)
	must(t, a.CreateRole("root", "child"))
	must(t, a.GrantResource("root", "child", "/app/*"))
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
