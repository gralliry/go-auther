package auther

import (
	"testing"

	"github.com/gralliry/go-auther/adapter/driver/noop"
	"github.com/gralliry/go-auther/errors"
)

var (
	ErrRoleNotFound       = errors.ErrRoleNotFound
	ErrPolicyNotFound     = errors.ErrPolicyNotFound
	ErrRoleSelfGrant      = errors.ErrRoleSelfGrant
	ErrRoleInsufficient   = errors.ErrRoleInsufficient
	ErrRoleAlreadyAssigned = errors.ErrRoleAlreadyAssigned
	ErrRoleNotAssigned    = errors.ErrRoleNotAssigned
	ErrUserNotFound       = errors.ErrUserNotFound
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	m, err := NewManager(noop.New())
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// must fails the test if err is non-nil.
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// =============================================================================
// Grant tests
// =============================================================================

func TestGrant(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)

		must(t, m.CreateRole("admin"))
		must(t, m.Grant("root", "/user/*", "admin"))

		ok, err := m.EnforceByRole("admin", "/user/create")
		must(t, err)
		if !ok {
			t.Error("admin should have access to /user/create")
		}

		ok, err = m.EnforceByRole("admin", "/data/read")
		must(t, err)
		if ok {
			t.Error("admin should NOT have access to /data/read")
		}
	})

	t.Run("multilevel", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))
		must(t, m.CreateRole("editor"))

		must(t, m.Grant("root", "/user/*", "admin"))
		must(t, m.Grant("admin", "/user/profile", "editor"))

		ok, _ := m.EnforceByRole("editor", "/user/profile")
		if !ok {
			t.Error("editor should have access to /user/profile")
		}

		ok, _ = m.EnforceByRole("editor", "/user/create")
		if ok {
			t.Error("editor should NOT have access to /user/create")
		}
	})

	t.Run("narrower", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))
		must(t, m.CreateRole("editor"))

		must(t, m.Grant("root", "/user/*", "admin"))
		must(t, m.Grant("admin", "/user/profile", "editor"))

		ok, _ := m.EnforceByRole("editor", "/user/profile")
		if !ok {
			t.Error("editor should have /user/profile")
		}
		ok, _ = m.EnforceByRole("editor", "/user/create")
		if ok {
			t.Error("editor should NOT have /user/create (only has /user/profile)")
		}
	})

	t.Run("exactVsWildcard", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))

		must(t, m.Grant("root", "/user/create", "admin"))

		ok, _ := m.EnforceByRole("admin", "/user/create")
		if !ok {
			t.Error("admin should have exact match")
		}
		ok, _ = m.EnforceByRole("admin", "/user/delete")
		if ok {
			t.Error("admin should NOT have /user/delete")
		}
	})

	t.Run("rootWildcard", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)

		ok, _ := m.EnforceByRole("root", "/anything")
		if !ok {
			t.Error("root should match /**")
		}
		ok, _ = m.EnforceByRole("root", "/deep/nested/path")
		if !ok {
			t.Error("root should match any path via /**")
		}
	})

	t.Run("normalization", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))

		must(t, m.Grant("root", "user/create", "admin"))
		ok, _ := m.EnforceByRole("admin", "/user/create")
		if !ok {
			t.Error("no-slash path should be normalized to /user/create")
		}

		must(t, m.Grant("root", "/user//profile", "admin"))
		ok, _ = m.EnforceByRole("admin", "/user/profile")
		if !ok {
			t.Error("double-slash should be normalized")
		}
	})
}

// =============================================================================
// Revoke tests
// =============================================================================

func TestRevoke(t *testing.T) {
	t.Run("single", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))

		must(t, m.Grant("root", "/user/*", "admin"))
		ok, _ := m.EnforceByRole("admin", "/user/create")
		if !ok {
			t.Fatal("admin should have access before revoke")
		}

		must(t, m.Revoke("root", "/user/*"))

		ok, _ = m.EnforceByRole("admin", "/user/create")
		if ok {
			t.Error("admin should NOT have access after revoke")
		}
	})

	t.Run("cascade", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))
		must(t, m.CreateRole("editor"))

		must(t, m.Grant("root", "/user/*", "admin"))
		must(t, m.Grant("admin", "/user/profile", "editor"))

		ok, _ := m.EnforceByRole("editor", "/user/profile")
		if !ok {
			t.Fatal("editor should have access before revoke")
		}

		m.Revoke("root", "/user/*")

		ok, _ = m.EnforceByRole("admin", "/user/create")
		if ok {
			t.Error("admin should NOT have access after revoke")
		}
		ok, _ = m.EnforceByRole("editor", "/user/profile")
		if ok {
			t.Error("editor should NOT have access after cascade revoke")
		}
	})

	t.Run("deepCascade", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("a"))
		must(t, m.CreateRole("b"))
		must(t, m.CreateRole("c"))

		must(t, m.Grant("root", "/data/**", "a"))
		must(t, m.Grant("a", "/data/reports/*", "b"))
		must(t, m.Grant("b", "/data/reports/q1", "c"))

		ok, _ := m.EnforceByRole("c", "/data/reports/q1")
		if !ok {
			t.Fatal("c should have access before revoke")
		}

		m.Revoke("root", "/data/**")

		ok, _ = m.EnforceByRole("c", "/data/reports/q1")
		if ok {
			t.Error("c should NOT have access after deep cascade revoke")
		}
	})

	t.Run("independentBranch", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))
		must(t, m.CreateRole("editor"))

		must(t, m.Grant("root", "/user/*", "admin"))
		must(t, m.Grant("root", "/data/*", "editor"))
		must(t, m.Grant("admin", "/user/profile", "editor"))

		m.Revoke("root", "/user/*")

		ok, _ := m.EnforceByRole("editor", "/user/profile")
		if ok {
			t.Error("editor should NOT have /user/profile after cascade")
		}

		ok, _ = m.EnforceByRole("editor", "/data/read")
		if !ok {
			t.Error("editor should still have /data/read (independent branch)")
		}
	})

	t.Run("dagMultiParent", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("A"))
		must(t, m.CreateRole("B"))
		must(t, m.CreateRole("C"))
		must(t, m.CreateRole("D"))

		must(t, m.Grant("root", "/a/*", "A"))
		must(t, m.Grant("A", "/a/*", "B"))
		must(t, m.Grant("A", "/a/*", "C"))
		must(t, m.Grant("C", "/a/b", "D"))
		must(t, m.Grant("B", "/a/*", "C"))

		ok, _ := m.EnforceByRole("D", "/a/b")
		if !ok {
			t.Fatal("D should have /a/b before revoke")
		}

		m.Revoke("A", "/a/*")

		ok, _ = m.EnforceByRole("C", "/a/b")
		if !ok {
			t.Error("C should still have /a/b via B after A revokes")
		}

		ok, _ = m.EnforceByRole("D", "/a/b")
		if !ok {
			t.Error("D should still have /a/b after partial revoke")
		}
	})
}

// =============================================================================
// Error-path tests
// =============================================================================

func TestErrors(t *testing.T) {
	t.Run("selfGrant", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		err := m.Grant("root", "/user/*", "root")
		if err != ErrRoleSelfGrant {
			t.Errorf("expected ErrRoleSelfGrant, got %v", err)
		}
	})

	t.Run("insufficient", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))
		must(t, m.CreateRole("editor"))
		must(t, m.Grant("root", "/user/*", "admin"))

		err := m.Grant("admin", "/data/*", "editor")
		if err != ErrRoleInsufficient {
			t.Errorf("expected ErrRoleInsufficient, got %v", err)
		}
	})

	t.Run("revokeNotFound", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))
		must(t, m.Grant("root", "/user/*", "admin"))

		err := m.Revoke("admin", "/user/*")
		if err != ErrPolicyNotFound {
			t.Errorf("expected ErrPolicyNotFound, got %v", err)
		}
	})

	t.Run("narrowerGrantWider", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))
		must(t, m.CreateRole("editor"))
		must(t, m.Grant("root", "/user/profile", "admin"))

		err := m.Grant("admin", "/user/*", "editor")
		if err != ErrRoleInsufficient {
			t.Errorf("expected ErrRoleInsufficient, got %v", err)
		}
	})

	t.Run("grantToDeleted", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))
		must(t, m.CreateRole("editor"))
		must(t, m.Grant("root", "/user/*", "admin"))
		must(t, m.DeleteRole("editor"))

		err := m.Grant("admin", "/user/profile", "editor")
		if err != ErrRoleNotFound {
			t.Errorf("expected ErrRoleNotFound, got %v", err)
		}
	})

	t.Run("enforceDeleted", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))
		must(t, m.Grant("root", "/user/*", "admin"))
		must(t, m.DeleteRole("admin"))

		_, err := m.EnforceByRole("admin", "/user/create")
		if err != ErrRoleNotFound {
			t.Errorf("expected ErrRoleNotFound, got %v", err)
		}
	})

	t.Run("duplicateRole", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		err := m.CreateRole("root")
		if err == nil {
			t.Error("expected error for duplicate role")
		}
	})

	t.Run("duplicateUser", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateUser("bob"))
		err := m.CreateUser("bob")
		if err == nil {
			t.Error("expected error for duplicate user")
		}
	})

	t.Run("deleteTwiceRole", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))
		must(t, m.DeleteRole("admin"))

		err := m.DeleteRole("admin")
		if err != ErrRoleNotFound {
			t.Errorf("expected ErrRoleNotFound on second delete, got %v", err)
		}
	})

	t.Run("deleteTwiceUser", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateUser("alice"))
		must(t, m.DeleteUser("alice"))

		err := m.DeleteUser("alice")
		if err != ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound on second delete, got %v", err)
		}
	})

	t.Run("alreadyAssigned", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))
		must(t, m.CreateUser("alice"))
		must(t, m.Assign("alice", "admin"))

		err := m.Assign("alice", "admin")
		if err != ErrRoleAlreadyAssigned {
			t.Errorf("expected ErrRoleAlreadyAssigned, got %v", err)
		}
	})

	t.Run("notAssigned", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))
		must(t, m.CreateUser("alice"))

		err := m.Unassign("alice", "admin")
		if err != ErrRoleNotAssigned {
			t.Errorf("expected ErrRoleNotAssigned, got %v", err)
		}
	})

	t.Run("deletedRoleAssign", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))
		must(t, m.CreateUser("alice"))
		must(t, m.DeleteRole("admin"))

		err := m.Assign("alice", "admin")
		if err != ErrRoleNotFound {
			t.Errorf("expected ErrRoleNotFound, got %v", err)
		}
	})

	t.Run("assignedAfterDelete", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))
		must(t, m.CreateUser("alice"))
		must(t, m.DeleteUser("alice"))

		_, err := m.IsAssigned("alice", "admin")
		if err != ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})
}

// =============================================================================
// User tests
// =============================================================================

func TestUser(t *testing.T) {
	t.Run("assignAndEnforce", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))
		must(t, m.Grant("root", "/user/*", "admin"))
		must(t, m.CreateUser("alice"))
		must(t, m.Assign("alice", "admin"))

		ok, _ := m.EnforceByUser("alice", "/user/create")
		if !ok {
			t.Error("alice should have access to /user/create via admin role")
		}

		ok, _ = m.EnforceByUser("alice", "/data/read")
		if ok {
			t.Error("alice should NOT have access to /data/read")
		}
	})

	t.Run("unassign", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))
		must(t, m.Grant("root", "/user/*", "admin"))
		must(t, m.CreateUser("alice"))
		must(t, m.Assign("alice", "admin"))
		must(t, m.Unassign("alice", "admin"))

		ok, _ := m.EnforceByUser("alice", "/user/create")
		if ok {
			t.Error("alice should NOT have access after unassign")
		}
	})

	t.Run("delete", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))
		must(t, m.Grant("root", "/user/*", "admin"))
		must(t, m.CreateUser("alice"))
		must(t, m.Assign("alice", "admin"))
		must(t, m.DeleteUser("alice"))

		if m.CheckUser("alice") {
			t.Error("alice should be gone after delete")
		}

		_, err := m.EnforceByUser("alice", "/user/create")
		if err != ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})

	t.Run("multipleRoles", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))
		must(t, m.CreateRole("editor"))
		must(t, m.Grant("root", "/user/*", "admin"))
		must(t, m.Grant("root", "/data/*", "editor"))
		must(t, m.CreateUser("alice"))
		must(t, m.Assign("alice", "admin"))
		must(t, m.Assign("alice", "editor"))

		ok, _ := m.EnforceByUser("alice", "/user/create")
		if !ok {
			t.Error("alice should have /user/create via admin")
		}

		ok, _ = m.EnforceByUser("alice", "/data/read")
		if !ok {
			t.Error("alice should have /data/read via editor")
		}

		ok, _ = m.EnforceByUser("alice", "/reports/q1")
		if ok {
			t.Error("alice should NOT have /reports/q1 (no role grants it)")
		}
	})

	t.Run("isAssigned", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))
		must(t, m.CreateUser("alice"))

		ok, _ := m.IsAssigned("alice", "admin")
		if ok {
			t.Error("alice should not have admin assigned yet")
		}

		must(t, m.Assign("alice", "admin"))

		ok, _ = m.IsAssigned("alice", "admin")
		if !ok {
			t.Error("alice should have admin assigned")
		}
	})

	t.Run("noRolesEnforce", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateUser("alice"))

		ok, _ := m.EnforceByUser("alice", "/anything")
		if ok {
			t.Error("alice with no roles should not have access")
		}
	})
}

// =============================================================================
// Role lifecycle tests
// =============================================================================

func TestRoleLifecycle(t *testing.T) {
	t.Run("deleteCascade", func(t *testing.T) {
		t.Parallel()
		m := newTestManager(t)
		must(t, m.CreateRole("admin"))
		must(t, m.CreateRole("editor"))

		must(t, m.Grant("root", "/user/*", "admin"))
		must(t, m.Grant("admin", "/user/profile", "editor"))

		ok, _ := m.EnforceByRole("editor", "/user/profile")
		if !ok {
			t.Fatal("editor should have access before delete")
		}

		must(t, m.DeleteRole("admin"))

		_, err := m.EnforceByRole("admin", "/user/profile")
		if err == nil {
			t.Error("admin should be gone after delete")
		}

		ok, _ = m.EnforceByRole("editor", "/user/profile")
		if ok {
			t.Error("editor should NOT have access after admin deleted")
		}
	})
}
