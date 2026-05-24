package fileadapter

import (
	"os"
	"testing"

	"auther"
)

func TestFileAdapterRoundTrip(t *testing.T) {
	tmpFile := "test_policy.json"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + ".tmp")

	// Create authorizer with file adapter
	a1, err := auther.NewAuthorizer(NewFileAdapter(tmpFile))
	if err != nil {
		t.Fatalf("NewAuthorizer: %v", err)
	}

	must(t, a1.CreateRole("root", "admin"))
	must(t, a1.CreateRole("admin", "editor"))
	must(t, a1.GrantResource("admin", "admin", "/user/*"))
	must(t, a1.GrantResource("editor", "editor", "/data/*"))
	must(t, a1.GrantResource("root", "admin", "/g/**"))
	must(t, a1.CreateUser("editor", "editor_user"))

	// Reload from file
	a2, err := auther.NewAuthorizer(NewFileAdapter(tmpFile))
	if err != nil {
		t.Fatalf("NewAuthorizer reload: %v", err)
	}

	u, err := a2.GetUser("editor_user")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if u.RoleID != "editor" {
		t.Errorf("expected role editor, got %s", u.RoleID)
	}

	grants := a2.GetAllGrants()
	if len(grants) != 1 {
		t.Errorf("expected 1 grant, got %d", len(grants))
	}

	// Verify enforcement works (explicit-only model)
	ok, _ := a2.Enforce("editor_user", "/data/read")
	if !ok {
		t.Error("editor_user should have /data/read via editor role")
	}
	ok, _ = a2.Enforce("editor_user", "/anything")
	if ok {
		t.Error("editor_user should not have /** — no inheritance from root")
	}
}

func TestFileAdapterEmptyFile(t *testing.T) {
	tmpFile := "test_empty.json"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + ".tmp")

	a, err := auther.NewAuthorizer(NewFileAdapter(tmpFile))
	if err != nil {
		t.Fatalf("NewAuthorizer: %v", err)
	}

	// Should have auto-created root
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
