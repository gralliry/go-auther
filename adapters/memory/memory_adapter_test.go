package memory

import (
	"errors"
	"testing"

	"github.com/gralliry/go-auther"
)

func TestMemoryAdapterRoundTrip(t *testing.T) {
	adapter := New()

	a1, err := auther.NewAuthorizer(adapter)
	if err != nil {
		t.Fatalf("NewAuthorizer: %v", err)
	}

	must(t, a1.CreateRole("root", "editor"))
	must(t, a1.CreateUser("editor", "u1"))
	must(t, a1.Grant("root", "editor", "/data/*"))

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

	ok, _ := a2.Enforce("u1", "/anything")
	if ok {
		t.Error("user should NOT have /** access — no auto-inheritance")
	}

	ok, _ = a2.Enforce("u1", "/data/read")
	if !ok {
		t.Error("user should have /data/* from grant")
	}

	grants, _ := a2.GetGrantsFrom("root")
	if len(grants) != 1 {
		t.Errorf("expected 1 grant, got %d", len(grants))
	}
}

func TestMemoryAdapterEmptyStart(t *testing.T) {
	adapter := New()

	a, err := auther.NewAuthorizer(adapter)
	if err != nil {
		t.Fatalf("NewAuthorizer: %v", err)
	}

	ok, err := a.Enforce("nonexistent", "/anything")
	if !errors.Is(err, auther.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
	if ok {
		t.Error("expected false for nonexistent user")
	}

	role, err := a.GetRole("root")
	if err != nil {
		t.Fatal(err)
	}
	if len(role.Resources) != 1 || role.Resources[0] != "/**" {
		t.Errorf("expected root with /** resource, got %v", role.Resources)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
