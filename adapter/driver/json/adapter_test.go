package json

import (
	"os"
	"testing"

	"github.com/gralliry/go-auther/adapter"
)

func tempPath(t *testing.T) string {
	t.Helper()
	return t.TempDir() + "/policy.json"
}

func TestNew(t *testing.T) {
	t.Run("createsFile", func(t *testing.T) {
		a, err := New(tempPath(t))
		if err != nil {
			t.Fatal(err)
		}
		snap, err := a.All()
		if err != nil {
			t.Fatal(err)
		}
		if len(snap.Role) != 0 || len(snap.User) != 0 || len(snap.Policy) != 0 {
			t.Error("empty snapshot expected for new file")
		}
	})

	t.Run("loadsExisting", func(t *testing.T) {
		p := tempPath(t)

		// Write data.
		a1, _ := New(p)
		a1.CreateRole(adapter.Role{ID: "root"})
		a1.CreateUser(adapter.User{ID: "alice", RoleID: "root"})
		a1.CreatePolicy(adapter.Policy{ID: 1, GrantorRoleID: "root", GranteeRoleID: "admin", Resource: "/user/*"})

		// Reload.
		a2, err := New(p)
		if err != nil {
			t.Fatal(err)
		}
		snap, _ := a2.All()
		if len(snap.Role) != 1 || snap.Role[0].ID != "root" {
			t.Errorf("expected 1 role 'root', got %+v", snap.Role)
		}
		if len(snap.User) != 1 || snap.User[0].ID != "alice" {
			t.Errorf("expected 1 user 'alice', got %+v", snap.User)
		}
		if len(snap.Policy) != 1 || snap.Policy[0].Resource != "/user/*" {
			t.Errorf("expected 1 policy '/user/*', got %+v", snap.Policy)
		}
	})

	t.Run("atomicWriteNoCorruption", func(t *testing.T) {
		p := tempPath(t)
		_, err := New(p)
		if err != nil {
			t.Fatal(err)
		}

		// Simulate crashed write: leave a .tmp file behind.
		os.WriteFile(p+".tmp", []byte("garbage"), 0o644)

		// Load should ignore the .tmp.
		a2, err := New(p)
		if err != nil {
			t.Fatal(err)
		}
		snap, _ := a2.All()
		if len(snap.Role) != 0 {
			t.Error("expected empty snapshot, .tmp should not affect load")
		}
	})
}

func TestMutations(t *testing.T) {
	t.Run("createDuplicateNoop", func(t *testing.T) {
		a, _ := New(tempPath(t))
		a.CreateRole(adapter.Role{ID: "admin"})
		a.CreateRole(adapter.Role{ID: "admin"})

		snap, _ := a.All()
		if len(snap.Role) != 1 {
			t.Errorf("expected 1 role, got %d", len(snap.Role))
		}
	})

	t.Run("deleteRole", func(t *testing.T) {
		a, _ := New(tempPath(t))
		a.CreateRole(adapter.Role{ID: "admin"})
		a.DeleteRole(adapter.Role{ID: "admin"})

		snap, _ := a.All()
		if len(snap.Role) != 0 {
			t.Errorf("expected 0 roles after delete, got %d", len(snap.Role))
		}
	})

	t.Run("deleteUser", func(t *testing.T) {
		a, _ := New(tempPath(t))
		a.CreateUser(adapter.User{ID: "alice", RoleID: "root"})
		a.DeleteUser(adapter.User{ID: "alice"})

		snap, _ := a.All()
		if len(snap.User) != 0 {
			t.Errorf("expected 0 users after delete, got %d", len(snap.User))
		}
	})

	t.Run("deletePolicy", func(t *testing.T) {
		a, _ := New(tempPath(t))
		a.CreatePolicy(adapter.Policy{ID: 42, Resource: "/test"})
		a.DeletePolicy(42)

		snap, _ := a.All()
		if len(snap.Policy) != 0 {
			t.Errorf("expected 0 policies after delete, got %d", len(snap.Policy))
		}
	})

	t.Run("deleteNonexistentNoop", func(t *testing.T) {
		a, _ := New(tempPath(t))
		// These should not error.
		a.DeleteRole(adapter.Role{ID: "nonexistent"})
		a.DeleteUser(adapter.User{ID: "nonexistent"})
		a.DeletePolicy(9999)

		snap, _ := a.All()
		if len(snap.Role) != 0 || len(snap.User) != 0 || len(snap.Policy) != 0 {
			t.Error("deleting nonexistent entities should not create entries")
		}
	})
}

func TestConcurrency(t *testing.T) {
	a, _ := New(tempPath(t))

	done := make(chan struct{})
	for i := range 50 {
		go func(id int) {
			a.CreateRole(adapter.Role{ID: string(rune('A' + id%26))})
			a.All()
			done <- struct{}{}
		}(i)
	}
	for range 50 {
		<-done
	}
}
