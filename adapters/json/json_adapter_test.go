package json

import (
	"os"
	"testing"

	"github.com/gralliry/go-auther"
)

func TestJSONAdapterRoundTrip(t *testing.T) {
	tmpFile := "test_policy.json"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + ".tmp")

	a1, err := auther.NewAuthorizer(New(tmpFile))
	if err != nil {
		t.Fatalf("NewAuthorizer: %v", err)
	}

	must(t, a1.CreateRole("root", "admin"))
	must(t, a1.CreateRole("admin", "editor"))
	must(t, a1.Grant("root", "admin", "/user/*"))
	must(t, a1.Grant("root", "editor", "/data/*"))
	must(t, a1.Grant("root", "admin", "/g/**"))
	must(t, a1.CreateUser("editor", "editor_user"))

	a2, err := auther.NewAuthorizer(New(tmpFile))
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

	grants, _ := a2.GetGrantsFrom("root")
	if len(grants) != 3 {
		t.Errorf("expected 3 grants, got %d", len(grants))
	}

	ok, _ := a2.Enforce("editor_user", "/data/read")
	if !ok {
		t.Error("editor_user should have /data/read via editor role")
	}
	ok, _ = a2.Enforce("editor_user", "/anything")
	if ok {
		t.Error("editor_user should not have /** — no inheritance from root")
	}
}

func TestJSONAdapterEmptyFile(t *testing.T) {
	tmpFile := "test_empty.json"
	defer os.Remove(tmpFile)
	defer os.Remove(tmpFile + ".tmp")

	a, err := auther.NewAuthorizer(New(tmpFile))
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

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
