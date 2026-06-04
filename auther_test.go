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
