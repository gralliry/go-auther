package sql

import (
	"database/sql"
	"os"
	"testing"

	"github.com/gralliry/go-auther"

	_ "modernc.org/sqlite"
)

func TestSQLAdapterRoundTrip(t *testing.T) {
	db := openSQLite(t)
	defer db.Close()

	adapter, err := New(db, "", "auther_policy")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	a1, err := auther.NewAuthorizer(adapter)
	if err != nil {
		t.Fatalf("NewAuthorizer: %v", err)
	}

	must(t, a1.CreateRole("root", "admin"))
	must(t, a1.CreateRole("admin", "editor"))
	must(t, a1.Grant("root", "admin", "/user/*"))
	must(t, a1.Grant("root", "editor", "/data/*"))
	must(t, a1.Grant("root", "admin", "/g/**"))
	must(t, a1.CreateUser("editor", "u1"))

	a2, err := auther.NewAuthorizer(adapter)
	if err != nil {
		t.Fatalf("NewAuthorizer reload: %v", err)
	}

	u, err := a2.GetUser("u1")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if u.RoleID != "editor" {
		t.Errorf("expected role editor, got %s", u.RoleID)
	}

	grants, _ := a2.GetGrantsFrom("root")
	if len(grants) != 3 {
		t.Errorf("expected 3 grants, got %d", len(grants))
	}

	ok, _ := a2.Enforce("u1", "/data/read")
	if !ok {
		t.Error("user should have /data/read via editor role")
	}
	ok, _ = a2.Enforce("u1", "/anything")
	if ok {
		t.Error("user should not have /** — no inheritance from root")
	}
}

func TestSQLAdapterEmptyDB(t *testing.T) {
	db := openSQLite(t)
	defer db.Close()

	adapter, err := New(db, "", "policy")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	a, err := auther.NewAuthorizer(adapter)
	if err != nil {
		t.Fatalf("NewAuthorizer: %v", err)
	}

	role, err := a.GetRole("root")
	if err != nil {
		t.Fatal(err)
	}
	if len(role.Resources) != 1 || role.Resources[0] != "/**" {
		t.Errorf("expected root with /** resource, got %v", role.Resources)
	}
}

func TestSQLAdapterTablePrefix(t *testing.T) {
	db := openSQLite(t)
	defer db.Close()

	adapter, err := New(db, "pre_", "mytable")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	a1, err := auther.NewAuthorizer(adapter)
	if err != nil {
		t.Fatalf("NewAuthorizer: %v", err)
	}
	must(t, a1.CreateUser("root", "u"))

	a2, err := auther.NewAuthorizer(adapter)
	if err != nil {
		t.Fatalf("NewAuthorizer reload: %v", err)
	}
	_, err = a2.GetUser("u")
	if err != nil {
		t.Errorf("user should survive round-trip with prefix, got: %v", err)
	}
}

func TestSQLAdapterInvalidTableName(t *testing.T) {
	db := openSQLite(t)
	defer db.Close()

	bad := []string{"", "drop table", "1bad", "x; drop table"}
	for _, name := range bad {
		_, err := New(db, "", name)
		if err == nil {
			t.Errorf("expected error for table name %q", name)
		}
	}
}

func openSQLite(t *testing.T) *sql.DB {
	t.Helper()

	tmpFile := "test_sql_adapter.db"
	t.Cleanup(func() {
		os.Remove(tmpFile)
		os.Remove(tmpFile + "-wal")
		os.Remove(tmpFile + "-shm")
	})

	db, err := sql.Open("sqlite", tmpFile)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	return db
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
