package sql

import (
	"testing"

	"github.com/gralliry/go-auther/adapter"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	return db
}

func newTestAdapter(t *testing.T) *Adapter {
	t.Helper()
	a, err := New(newTestDB(t))
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	return a
}

func TestRoles(t *testing.T) {
	t.Parallel()
	a := newTestAdapter(t)

	if err := a.CreateRole(adapter.Role{ID: "admin"}); err != nil {
		t.Fatal(err)
	}
	if err := a.CreateRole(adapter.Role{ID: "admin"}); err != nil { // duplicate no-op
		t.Fatal(err)
	}
	if err := a.CreateRole(adapter.Role{ID: "editor"}); err != nil {
		t.Fatal(err)
	}

	snap, _ := a.Snapshot()
	if len(snap.Role) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(snap.Role))
	}

	if err := a.DeleteRole(adapter.Role{ID: "admin"}); err != nil {
		t.Fatal(err)
	}
	if err := a.DeleteRole(adapter.Role{ID: "ghost"}); err != nil { // missing no-op
		t.Fatal(err)
	}

	snap, _ = a.Snapshot()
	if len(snap.Role) != 1 || snap.Role[0].ID != "editor" {
		t.Fatalf("expected 1 role 'editor', got %+v", snap.Role)
	}
}

func TestUsers(t *testing.T) {
	t.Parallel()
	a := newTestAdapter(t)

	a.LinkUser(adapter.User{ID: "alice", RoleID: "admin"})
	a.LinkUser(adapter.User{ID: "alice", RoleID: "editor"})
	a.LinkUser(adapter.User{ID: "bob", RoleID: "viewer"})
	a.LinkUser(adapter.User{ID: "alice", RoleID: "admin"}) // duplicate no-op

	snap, _ := a.Snapshot()
	if len(snap.User) != 3 {
		t.Fatalf("expected 3 users, got %d", len(snap.User))
	}

	a.UnlinkUser(adapter.User{ID: "alice", RoleID: "editor"})
	if err := a.UnlinkUser(adapter.User{ID: "alice", RoleID: "ghost"}); err != nil { // missing no-op
		t.Fatal(err)
	}

	a.DeleteUser(adapter.User{ID: "bob"})
	if err := a.DeleteUser(adapter.User{ID: "ghost"}); err != nil { // missing no-op
		t.Fatal(err)
	}

	snap, _ = a.Snapshot()
	if len(snap.User) != 1 || snap.User[0].RoleID != "admin" {
		t.Fatalf("expected 1 user with role 'admin', got %+v", snap.User)
	}
}

func TestPolicies(t *testing.T) {
	t.Parallel()
	a := newTestAdapter(t)

	p1 := adapter.Policy{ID: 1, GrantorRoleID: "root", GranteeRoleID: "admin", Resource: "/user/*"}
	p2 := adapter.Policy{ID: 2, GrantorRoleID: "admin", GranteeRoleID: "editor", Resource: "/data/**"}

	a.CreatePolicy(p1)
	a.CreatePolicy(p2)
	a.CreatePolicy(p1) // duplicate no-op

	snap, _ := a.Snapshot()
	if len(snap.Policy) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(snap.Policy))
	}

	a.DeletePolicy(1)
	if err := a.DeletePolicy(999); err != nil { // missing no-op
		t.Fatal(err)
	}

	snap, _ = a.Snapshot()
	if len(snap.Policy) != 1 || snap.Policy[0].ID != 2 {
		t.Fatalf("expected 1 policy (id=2), got %+v", snap.Policy)
	}
}

func TestEmptySnapshot(t *testing.T) {
	t.Parallel()
	a := newTestAdapter(t)

	snap, err := a.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Role) != 0 || len(snap.User) != 0 || len(snap.Policy) != 0 {
		t.Fatal("expected empty snapshot")
	}
}
