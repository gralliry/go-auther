package sql

import (
	"testing"

	"github.com/gralliry/go-auther"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSQLAdapterRoundTrip(t *testing.T) {
	db := openSQLite(t)

	adapter := New(db)

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

	adapter := New(db)

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

func openSQLite(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(t.TempDir()+"/test.db"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	t.Cleanup(func() {
		if sqldb, err := db.DB(); err == nil {
			sqldb.Close()
		}
	})
	return db
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
