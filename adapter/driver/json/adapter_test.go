package json

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gralliry/go-auther/adapter"
)

func tempPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "policy.json")
}

func TestNewCreatesFile(t *testing.T) {
	p := tempPath(t)
	a, err := New(p)
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
}

func TestNewLoadsExistingFile(t *testing.T) {
	p := tempPath(t)

	// First adapter: write data
	a1, _ := New(p)
	a1.CreateRole(adapter.Role{ID: "root"})
	a1.CreateUser(adapter.User{ID: "alice", RoleID: "root"})
	a1.CreatePolicy(adapter.Policy{ID: 1, GrantorRoleID: "root", GranteeRoleID: "admin", Resource: "/user/*"})

	// Second adapter: should load the persisted data
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
}

func TestCreateDuplicateNoop(t *testing.T) {
	a, _ := New(tempPath(t))
	a.CreateRole(adapter.Role{ID: "admin"})
	a.CreateRole(adapter.Role{ID: "admin"})

	snap, _ := a.All()
	if len(snap.Role) != 1 {
		t.Errorf("expected 1 role, got %d", len(snap.Role))
	}
}

func TestDeleteRole(t *testing.T) {
	a, _ := New(tempPath(t))
	a.CreateRole(adapter.Role{ID: "admin"})
	a.DeleteRole("admin")

	snap, _ := a.All()
	if len(snap.Role) != 0 {
		t.Errorf("expected 0 roles after delete, got %d", len(snap.Role))
	}
}

func TestDeleteUser(t *testing.T) {
	a, _ := New(tempPath(t))
	a.CreateUser(adapter.User{ID: "alice", RoleID: "root"})
	a.DeleteUser("alice")

	snap, _ := a.All()
	if len(snap.User) != 0 {
		t.Errorf("expected 0 users after delete, got %d", len(snap.User))
	}
}

func TestDeletePolicy(t *testing.T) {
	a, _ := New(tempPath(t))
	a.CreatePolicy(adapter.Policy{ID: 42, Resource: "/test"})
	a.DeletePolicy(42)

	snap, _ := a.All()
	if len(snap.Policy) != 0 {
		t.Errorf("expected 0 policies after delete, got %d", len(snap.Policy))
	}
}

func TestAtomicWriteNoCorruption(t *testing.T) {
	p := tempPath(t)
	// Create adapter and save a valid snapshot.
	_, err := New(p)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate a crashed write: leave a .tmp file behind.
	os.WriteFile(p+".tmp", []byte("garbage"), 0o644)

	// Load should still work from the last successful save.
	a2, err := New(p)
	if err != nil {
		t.Fatal(err)
	}
	snap, _ := a2.All()
	// The garbage .tmp should be ignored; data is from original save.
	if len(snap.Role) != 0 {
		t.Error("expected empty snapshot, .tmp should not affect load")
	}
}

func TestConcurrency(t *testing.T) {
	a, _ := New(tempPath(t))

	done := make(chan struct{})
	for i := 0; i < 50; i++ {
		go func(id int) {
			a.CreateRole(adapter.Role{ID: string(rune('A' + id%26))})
			a.All()
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < 50; i++ {
		<-done
	}
}
