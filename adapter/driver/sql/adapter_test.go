package sql

import (
	"database/sql"
	"testing"

	"github.com/gralliry/go-auther/adapter"

	_ "modernc.org/sqlite"
)

// newTestDB opens an in-memory SQLite database for testing.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// newTestAdapter creates an adapter backed by an in-memory SQLite database.
func newTestAdapter(t *testing.T) *Adapter {
	t.Helper()
	a, err := New(newTestDB(t))
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	return a
}

// =============================================================================
// Role tests
// =============================================================================

func TestRoleCreateAndLoad(t *testing.T) {
	t.Parallel()
	a := newTestAdapter(t)

	if err := a.CreateRole(adapter.Role{ID: "admin"}); err != nil {
		t.Fatal(err)
	}
	if err := a.CreateRole(adapter.Role{ID: "editor"}); err != nil {
		t.Fatal(err)
	}

	snap, err := a.All()
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Role) != 2 {
		t.Fatalf("got %d roles, want 2", len(snap.Role))
	}
}

func TestRoleCreateDuplicateIgnored(t *testing.T) {
	t.Parallel()
	a := newTestAdapter(t)

	if err := a.CreateRole(adapter.Role{ID: "admin"}); err != nil {
		t.Fatal(err)
	}
	// Second insert should be silently ignored.
	if err := a.CreateRole(adapter.Role{ID: "admin"}); err != nil {
		t.Fatal(err)
	}

	snap, _ := a.All()
	if len(snap.Role) != 1 {
		t.Fatalf("got %d roles, want 1", len(snap.Role))
	}
}

func TestRoleDelete(t *testing.T) {
	t.Parallel()
	a := newTestAdapter(t)

	a.CreateRole(adapter.Role{ID: "admin"})
	a.CreateRole(adapter.Role{ID: "editor"})
	a.DeleteRole(adapter.Role{ID: "admin"})

	snap, _ := a.All()
	if len(snap.Role) != 1 {
		t.Fatalf("got %d roles, want 1", len(snap.Role))
	}
	if snap.Role[0].ID != "editor" {
		t.Fatalf("remaining role is %q, want editor", snap.Role[0].ID)
	}
}

func TestRoleDeleteMissing(t *testing.T) {
	t.Parallel()
	a := newTestAdapter(t)
	// Deleting a non-existent role should not error.
	if err := a.DeleteRole(adapter.Role{ID: "ghost"}); err != nil {
		t.Fatal(err)
	}
}

// =============================================================================
// User tests
// =============================================================================

func TestUserCreateAndLoad(t *testing.T) {
	t.Parallel()
	a := newTestAdapter(t)

	a.CreateUser(adapter.User{ID: "alice", RoleID: "admin"})
	a.CreateUser(adapter.User{ID: "alice", RoleID: "editor"})
	a.CreateUser(adapter.User{ID: "bob", RoleID: "viewer"})

	snap, _ := a.All()
	if len(snap.User) != 3 {
		t.Fatalf("got %d users, want 3", len(snap.User))
	}
}

func TestUserCreateDuplicateIgnored(t *testing.T) {
	t.Parallel()
	a := newTestAdapter(t)

	a.CreateUser(adapter.User{ID: "alice", RoleID: "admin"})
	// Insert same (ID, RoleID) pair — silently ignored.
	a.CreateUser(adapter.User{ID: "alice", RoleID: "admin"})

	snap, _ := a.All()
	if len(snap.User) != 1 {
		t.Fatalf("got %d users, want 1", len(snap.User))
	}
}

func TestUserDeleteAllRecords(t *testing.T) {
	t.Parallel()
	a := newTestAdapter(t)

	a.CreateUser(adapter.User{ID: "alice", RoleID: "admin"})
	a.CreateUser(adapter.User{ID: "alice", RoleID: "editor"})
	a.CreateUser(adapter.User{ID: "bob", RoleID: "viewer"})

	a.DeleteUser(adapter.User{ID: "alice"})

	snap, _ := a.All()
	if len(snap.User) != 1 {
		t.Fatalf("got %d users, want 1", len(snap.User))
	}
	if snap.User[0].ID != "bob" {
		t.Fatalf("remaining user is %q, want bob", snap.User[0].ID)
	}
}

func TestUserDeleteMissing(t *testing.T) {
	t.Parallel()
	a := newTestAdapter(t)
	if err := a.DeleteUser(adapter.User{ID: "ghost"}); err != nil {
		t.Fatal(err)
	}
}

func TestUserUnassign(t *testing.T) {
	t.Parallel()
	a := newTestAdapter(t)

	a.CreateUser(adapter.User{ID: "alice", RoleID: "admin"})
	a.CreateUser(adapter.User{ID: "alice", RoleID: "editor"})

	a.UnassignUser(adapter.User{ID: "alice", RoleID: "admin"})

	snap, _ := a.All()
	if len(snap.User) != 1 {
		t.Fatalf("got %d users after unassign, want 1", len(snap.User))
	}
	if snap.User[0].RoleID != "editor" {
		t.Fatalf("remaining role is %q, want editor", snap.User[0].RoleID)
	}
}

func TestUserUnassignMissing(t *testing.T) {
	t.Parallel()
	a := newTestAdapter(t)
	// Unassign non-existent pair — no-op.
	if err := a.UnassignUser(adapter.User{ID: "alice", RoleID: "ghost"}); err != nil {
		t.Fatal(err)
	}
}

// =============================================================================
// Policy tests
// =============================================================================

func TestPolicyCreateAndLoad(t *testing.T) {
	t.Parallel()
	a := newTestAdapter(t)

	p1 := adapter.Policy{ID: 1, GrantorRoleID: "root", GranteeRoleID: "admin", Resource: "/user/*"}
	p2 := adapter.Policy{ID: 2, GrantorRoleID: "admin", GranteeRoleID: "editor", Resource: "/data/**"}

	a.CreatePolicy(p1)
	a.CreatePolicy(p2)

	snap, _ := a.All()
	if len(snap.Policy) != 2 {
		t.Fatalf("got %d policies, want 2", len(snap.Policy))
	}
}

func TestPolicyCreateDuplicateIgnored(t *testing.T) {
	t.Parallel()
	a := newTestAdapter(t)

	p := adapter.Policy{ID: 1, GrantorRoleID: "root", GranteeRoleID: "admin", Resource: "/user/*"}
	a.CreatePolicy(p)
	a.CreatePolicy(p) // second insert silently ignored

	snap, _ := a.All()
	if len(snap.Policy) != 1 {
		t.Fatalf("got %d policies, want 1", len(snap.Policy))
	}
}

func TestPolicyDelete(t *testing.T) {
	t.Parallel()
	a := newTestAdapter(t)

	a.CreatePolicy(adapter.Policy{ID: 1, GrantorRoleID: "root", GranteeRoleID: "admin", Resource: "/user/*"})
	a.CreatePolicy(adapter.Policy{ID: 2, GrantorRoleID: "root", GranteeRoleID: "editor", Resource: "/data/**"})

	a.DeletePolicy(1)

	snap, _ := a.All()
	if len(snap.Policy) != 1 {
		t.Fatalf("got %d policies, want 1", len(snap.Policy))
	}
	if snap.Policy[0].ID != 2 {
		t.Fatalf("remaining policy id is %d, want 2", snap.Policy[0].ID)
	}
}

func TestPolicyDeleteMissing(t *testing.T) {
	t.Parallel()
	a := newTestAdapter(t)
	if err := a.DeletePolicy(999); err != nil {
		t.Fatal(err)
	}
}

// =============================================================================
// Snapshot / round-trip tests
// =============================================================================

func TestAllEmptySnapshot(t *testing.T) {
	t.Parallel()
	a := newTestAdapter(t)

	snap, err := a.All()
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Role) != 0 || len(snap.User) != 0 || len(snap.Policy) != 0 {
		t.Fatal("expected empty snapshot")
	}
}

func TestFullSnapshotRoundTrip(t *testing.T) {
	t.Parallel()
	a := newTestAdapter(t)

	// Populate all three tables.
	a.CreateRole(adapter.Role{ID: "admin"})
	a.CreateRole(adapter.Role{ID: "editor"})
	a.CreateUser(adapter.User{ID: "alice", RoleID: "admin"})
	a.CreateUser(adapter.User{ID: "alice", RoleID: "editor"})
	a.CreatePolicy(adapter.Policy{ID: 1, GrantorRoleID: "root", GranteeRoleID: "admin", Resource: "/**"})

	snap, err := a.All()
	if err != nil {
		t.Fatal(err)
	}

	if len(snap.Role) != 2 {
		t.Errorf("roles: got %d, want 2", len(snap.Role))
	}
	if len(snap.User) != 2 {
		t.Errorf("users: got %d, want 2", len(snap.User))
	}
	if len(snap.Policy) != 1 {
		t.Errorf("policies: got %d, want 1", len(snap.Policy))
	}

	// Verify values.
	found := false
	for _, r := range snap.Role {
		if r.ID == "admin" {
			found = true
			break
		}
	}
	if !found {
		t.Error("admin role not found in snapshot")
	}
}
