package auther

import (
	"testing"

	"github.com/gralliry/go-auther/adapter/empty"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	m, err := New(empty.New())
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestGrantAndEnforce(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")

	admin, err := m.CreateRole("admin")
	if err != nil {
		t.Fatal(err)
	}

	_, err = root.Grant(NewResource("/user/*"), admin)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := admin.Enforce(NewResource("/user/create"))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("admin should have access to /user/create")
	}

	ok, err = admin.Enforce(NewResource("/data/read"))
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("admin should NOT have access to /data/read")
	}
}

func TestMultiLevelDelegation(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")
	editor, _ := m.CreateRole("editor")

	root.Grant(NewResource("/user/*"), admin)
	admin.Grant(NewResource("/user/profile"), editor)

	ok, _ := editor.Enforce(NewResource("/user/profile"))
	if !ok {
		t.Error("editor should have access to /user/profile")
	}

	ok, _ = editor.Enforce(NewResource("/user/create"))
	if ok {
		t.Error("editor should NOT have access to /user/create")
	}
}

func TestRevokeSinglePolicy(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")

	policy, _ := root.Grant(NewResource("/user/*"), admin)
	ok, _ := admin.Enforce(NewResource("/user/create"))
	if !ok {
		t.Fatal("admin should have access before revoke")
	}

	err := root.Revoke(policy)
	if err != nil {
		t.Fatal(err)
	}

	ok, _ = admin.Enforce(NewResource("/user/create"))
	if ok {
		t.Error("admin should NOT have access after revoke")
	}
}

func TestRevokeCascade(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")
	editor, _ := m.CreateRole("editor")

	p, _ := root.Grant(NewResource("/user/*"), admin)
	admin.Grant(NewResource("/user/profile"), editor)

	ok, _ := editor.Enforce(NewResource("/user/profile"))
	if !ok {
		t.Fatal("editor should have access before revoke")
	}

	root.Revoke(p)

	ok, _ = admin.Enforce(NewResource("/user/create"))
	if ok {
		t.Error("admin should NOT have access after revoke")
	}
	ok, _ = editor.Enforce(NewResource("/user/profile"))
	if ok {
		t.Error("editor should NOT have access after cascade revoke")
	}
}

func TestRevokeDeepCascade(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")
	a, _ := m.CreateRole("a")
	b, _ := m.CreateRole("b")
	c, _ := m.CreateRole("c")

	p, _ := root.Grant(NewResource("/data/**"), a)
	a.Grant(NewResource("/data/reports/*"), b)
	b.Grant(NewResource("/data/reports/q1"), c)

	ok, _ := c.Enforce(NewResource("/data/reports/q1"))
	if !ok {
		t.Fatal("c should have access before revoke")
	}

	root.Revoke(p)

	ok, _ = c.Enforce(NewResource("/data/reports/q1"))
	if ok {
		t.Error("c should NOT have access after deep cascade revoke")
	}
}

func TestRoleDeleteCascade(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")
	editor, _ := m.CreateRole("editor")

	root.Grant(NewResource("/user/*"), admin)
	admin.Grant(NewResource("/user/profile"), editor)

	ok, _ := editor.Enforce(NewResource("/user/profile"))
	if !ok {
		t.Fatal("editor should have access before delete")
	}

	admin.Delete()

	if admin.Valid() {
		t.Error("admin should be invalid after delete")
	}

	ok, _ = editor.Enforce(NewResource("/user/profile"))
	if ok {
		t.Error("editor should NOT have access after admin deleted")
	}
}

func TestRevokeNotFound(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")

	p, _ := root.Grant(NewResource("/user/*"), admin)

	err := admin.Revoke(p)
	if err != ErrPolicyNotFound {
		t.Errorf("expected ErrPolicyNotFound, got %v", err)
	}
}

func TestSelfGrant(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")

	_, err := root.Grant(NewResource("/user/*"), root)
	if err != ErrRoleSelfGrant {
		t.Errorf("expected ErrRoleSelfGrant, got %v", err)
	}
}

func TestInsufficientPermission(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")
	editor, _ := m.CreateRole("editor")

	root.Grant(NewResource("/user/*"), admin)

	_, err := admin.Grant(NewResource("/data/*"), editor)
	if err != ErrRoleInsufficient {
		t.Errorf("expected ErrRoleInsufficient, got %v", err)
	}
}

func TestRevokeIndependentBranchUnaffected(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")
	editor, _ := m.CreateRole("editor")

	p1, _ := root.Grant(NewResource("/user/*"), admin)
	root.Grant(NewResource("/data/*"), editor)
	admin.Grant(NewResource("/user/profile"), editor)

	root.Revoke(p1)

	ok, _ := editor.Enforce(NewResource("/user/profile"))
	if ok {
		t.Error("editor should NOT have /user/profile after cascade")
	}

	ok, _ = editor.Enforce(NewResource("/data/read"))
	if !ok {
		t.Error("editor should still have /data/read (independent branch)")
	}
}

func TestEnforceDeletedRole(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")

	root.Grant(NewResource("/user/*"), admin)
	admin.Delete()

	_, err := admin.Enforce(NewResource("/user/create"))
	if err != ErrRoleInvalid {
		t.Errorf("expected ErrRoleInvalid, got %v", err)
	}
}

func TestDuplicateRole(t *testing.T) {
	m := newTestManager(t)
	_, err := m.CreateRole("root")
	if err == nil {
		t.Error("expected error for duplicate role")
	}
}

func TestUserAssignAndEnforce(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")

	root.Grant(NewResource("/user/*"), admin)

	alice, err := m.CreateUser("alice")
	if err != nil {
		t.Fatal(err)
	}
	alice.Assign(admin)

	ok, _ := alice.Enforce(NewResource("/user/create"))
	if !ok {
		t.Error("alice should have access to /user/create via admin role")
	}

	ok, _ = alice.Enforce(NewResource("/data/read"))
	if ok {
		t.Error("alice should NOT have access to /data/read")
	}
}

func TestUserUnassign(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")

	root.Grant(NewResource("/user/*"), admin)

	alice, _ := m.CreateUser("alice")
	alice.Assign(admin)

	alice.Unassign(admin)

	ok, _ := alice.Enforce(NewResource("/user/create"))
	if ok {
		t.Error("alice should NOT have access after unassign")
	}
}

func TestUserDelete(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")

	root.Grant(NewResource("/user/*"), admin)

	alice, _ := m.CreateUser("alice")
	alice.Assign(admin)
	alice.Delete()

	if alice.Valid() {
		t.Error("alice should be invalid after delete")
	}

	_, err := alice.Enforce(NewResource("/user/create"))
	if err != ErrUserInvalid {
		t.Errorf("expected ErrUserInvalid, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Additional tests
// ---------------------------------------------------------------------------

func TestGrantToDeletedRole(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")
	editor, _ := m.CreateRole("editor")

	root.Grant(NewResource("/user/*"), admin)
	editor.Delete()

	_, err := admin.Grant(NewResource("/user/profile"), editor)
	if err != ErrGranteeInvalid {
		t.Errorf("expected ErrGranteeInvalid, got %v", err)
	}
}

func TestGrantNarrowerDelegation(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")
	editor, _ := m.CreateRole("editor")

	// admin has /user/* from root
	root.Grant(NewResource("/user/*"), admin)
	// admin delegates a narrower subset to editor
	admin.Grant(NewResource("/user/profile"), editor)

	// editor can access /user/profile (within the delegated scope)
	ok, _ := editor.Enforce(NewResource("/user/profile"))
	if !ok {
		t.Error("editor should have /user/profile")
	}
	// editor cannot access a wider path (only has /user/profile, not /user/*)
	ok, _ = editor.Enforce(NewResource("/user/create"))
	if ok {
		t.Error("editor should NOT have /user/create (only has /user/profile)")
	}
}

func TestRootEnforceWildcard(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")

	ok, _ := root.Enforce(NewResource("/anything"))
	if !ok {
		t.Error("root should match /**")
	}
	ok, _ = root.Enforce(NewResource("/deep/nested/path"))
	if !ok {
		t.Error("root should match any path via /**")
	}
}

func TestUserMultipleRoles(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")
	editor, _ := m.CreateRole("editor")

	root.Grant(NewResource("/user/*"), admin)
	root.Grant(NewResource("/data/*"), editor)

	alice, _ := m.CreateUser("alice")
	alice.Assign(admin)
	alice.Assign(editor)

	ok, _ := alice.Enforce(NewResource("/user/create"))
	if !ok {
		t.Error("alice should have /user/create via admin")
	}

	ok, _ = alice.Enforce(NewResource("/data/read"))
	if !ok {
		t.Error("alice should have /data/read via editor")
	}

	ok, _ = alice.Enforce(NewResource("/reports/q1"))
	if ok {
		t.Error("alice should NOT have /reports/q1 (no role grants it)")
	}
}

func TestUserIsAssign(t *testing.T) {
	m := newTestManager(t)
	admin, _ := m.CreateRole("admin")
	alice, _ := m.CreateUser("alice")

	ok, _ := alice.IsAssign(admin)
	if ok {
		t.Error("alice should not have admin assigned yet")
	}

	alice.Assign(admin)

	ok, _ = alice.IsAssign(admin)
	if !ok {
		t.Error("alice should have admin assigned")
	}
}

func TestUserIsAssignInvalid(t *testing.T) {
	m := newTestManager(t)
	admin, _ := m.CreateRole("admin")
	alice, _ := m.CreateUser("alice")

	alice.Delete()

	_, err := alice.IsAssign(admin)
	if err != ErrUserInvalid {
		t.Errorf("expected ErrUserInvalid, got %v", err)
	}
}

func TestDuplicateUser(t *testing.T) {
	m := newTestManager(t)
	m.CreateUser("bob")
	_, err := m.CreateUser("bob")
	if err == nil {
		t.Error("expected error for duplicate user")
	}
}

func TestRoleDeleteTwice(t *testing.T) {
	m := newTestManager(t)
	admin, _ := m.CreateRole("admin")
	admin.Delete()

	err := admin.Delete()
	if err != ErrRoleInvalid {
		t.Errorf("expected ErrRoleInvalid on second delete, got %v", err)
	}
}

func TestUserDeleteTwice(t *testing.T) {
	m := newTestManager(t)
	alice, _ := m.CreateUser("alice")
	alice.Delete()

	err := alice.Delete()
	if err != ErrUserInvalid {
		t.Errorf("expected ErrUserInvalid on second delete, got %v", err)
	}
}

func TestRoleAlreadyAssigned(t *testing.T) {
	m := newTestManager(t)
	admin, _ := m.CreateRole("admin")
	alice, _ := m.CreateUser("alice")

	alice.Assign(admin)

	err := alice.Assign(admin)
	if err != ErrRoleAlreadyAssigned {
		t.Errorf("expected ErrRoleAlreadyAssigned, got %v", err)
	}
}

func TestRoleNotAssigned(t *testing.T) {
	m := newTestManager(t)
	admin, _ := m.CreateRole("admin")
	alice, _ := m.CreateUser("alice")

	err := alice.Unassign(admin)
	if err != ErrRoleNotAssigned {
		t.Errorf("expected ErrRoleNotAssigned, got %v", err)
	}
}

func TestRoleDeletedCannotBeAssigned(t *testing.T) {
	m := newTestManager(t)
	admin, _ := m.CreateRole("admin")
	alice, _ := m.CreateUser("alice")

	admin.Delete()

	err := alice.Assign(admin)
	if err != ErrRoleInvalid {
		t.Errorf("expected ErrRoleInvalid, got %v", err)
	}
}

func TestUserEnforceNoRoles(t *testing.T) {
	m := newTestManager(t)
	alice, _ := m.CreateUser("alice")

	ok, _ := alice.Enforce(NewResource("/anything"))
	if ok {
		t.Error("alice with no roles should not have access")
	}
}

func TestNarrowerCannotGrantWider(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")
	editor, _ := m.CreateRole("editor")

	root.Grant(NewResource("/user/profile"), admin)

	// admin has /user/profile but tries to grant /user/* (wider) to editor
	_, err := admin.Grant(NewResource("/user/*"), editor)
	if err != ErrRoleInsufficient {
		t.Errorf("expected ErrRoleInsufficient, got %v", err)
	}
}

func TestExactMatchVsWildcard(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")

	root.Grant(NewResource("/user/create"), admin)

	ok, _ := admin.Enforce(NewResource("/user/create"))
	if !ok {
		t.Error("admin should have exact match")
	}
	ok, _ = admin.Enforce(NewResource("/user/delete"))
	if ok {
		t.Error("admin should NOT have /user/delete")
	}
}

func TestResourceNormalization(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")
	admin, _ := m.CreateRole("admin")

	root.Grant(NewResource("user/create"), admin)
	ok, _ := admin.Enforce(NewResource("/user/create"))
	if !ok {
		t.Error("no-slash path should be normalized to /user/create")
	}

	root.Grant(NewResource("/user//profile"), admin)
	ok, _ = admin.Enforce(NewResource("/user/profile"))
	if !ok {
		t.Error("double-slash should be normalized")
	}
}

func TestDAGMultiParentSurvivesPartialRevoke(t *testing.T) {
	m := newTestManager(t)
	root, _ := m.GetRole("root")
	roleA, _ := m.CreateRole("A")
	roleB, _ := m.CreateRole("B")
	roleC, _ := m.CreateRole("C")
	roleD, _ := m.CreateRole("D")

	// A grants /a/* to B.
	root.Grant(NewResource("/a/*"), roleA)
	roleA.Grant(NewResource("/a/*"), roleB)

	// A grants /a/* to C.
	pAC, _ := roleA.Grant(NewResource("/a/*"), roleC)

	// C grants /a/b to D (narrower delegation).
	roleC.Grant(NewResource("/a/b"), roleD)

	// B also grants /a/* to C, so C now has /a/* from both A and B.
	roleB.Grant(NewResource("/a/*"), roleC)

	// Verify D has /a/b before revoke.
	ok, _ := roleD.Enforce(NewResource("/a/b"))
	if !ok {
		t.Fatal("D should have /a/b before revoke")
	}

	// A revokes its /a/* grant to C.
	roleA.Revoke(pAC)

	// C should still have /a/* via B.
	ok, _ = roleC.Enforce(NewResource("/a/b"))
	if !ok {
		t.Error("C should still have /a/b via B after A revokes")
	}

	// D should still have /a/b (C still has /a/* from B, so the delegation chain survives).
	ok, _ = roleD.Enforce(NewResource("/a/b"))
	if !ok {
		t.Error("D should still have /a/b after partial revoke")
	}
}
