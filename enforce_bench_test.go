package auther

import (
	"testing"

	"auther/match"
)

func BenchmarkMatchExact(b *testing.B) {
	for b.Loop() {
		match.Match("/user/create", "/user/create")
	}
}

func BenchmarkMatchNoMatchLiteral(b *testing.B) {
	for b.Loop() {
		match.Match("/user/create", "/user/delete")
	}
}

func BenchmarkMatchStar(b *testing.B) {
	for b.Loop() {
		match.Match("/user/*/edit", "/user/123/edit")
	}
}

func BenchmarkMatchDoubleStar(b *testing.B) {
	for b.Loop() {
		match.Match("/a/**/z", "/a/b/c/d/e/z")
	}
}

func BenchmarkMatchDoubleStarMany(b *testing.B) {
	for b.Loop() {
		match.Match("/api/**/export", "/api/v1/users/admin/reports/2024/export")
	}
}

// =============================================================================
// Enforce benchmarks — full enforcement path (role lookup + resource matching)
// =============================================================================

func benchAuthorizer(b *testing.B) *Authorizer {
	b.Helper()
	a, _ := NewAuthorizer(nil)
	_ = a.CreateRole("root", "admin")
	_ = a.CreateRole("admin", "editor")
	_ = a.Grant("admin", "admin", "/user/*")
	_ = a.Grant("editor", "editor", "/data/*")
	_ = a.Grant("root", "admin", "/g/**")
	_ = a.CreateUser("admin", "u_admin")
	_ = a.CreateUser("editor", "u_editor")
	return a
}

// Hit: exact match on role's own resource (fast path, no wildcard).
func BenchmarkEnforceHitExact(b *testing.B) {
	a := benchAuthorizer(b)
	b.ResetTimer()
	for b.Loop() {
		a.Enforce("u_editor", "/data/read")
	}
}

// Hit: wildcard match on role's own resource (needs segment parse + DP).
func BenchmarkEnforceHitWildcard(b *testing.B) {
	a := benchAuthorizer(b)
	b.ResetTimer()
	for b.Loop() {
		a.Enforce("u_admin", "/user/edit")
	}
}

// Hit: match via grant (grant patterns checked after own resources).
func BenchmarkEnforceHitGrant(b *testing.B) {
	a := benchAuthorizer(b)
	b.ResetTimer()
	for b.Loop() {
		a.Enforce("u_admin", "/g/anything/here")
	}
}

// Miss: no wildcards to check, first literal comparison fails immediately.
func BenchmarkEnforceMissLiteral(b *testing.B) {
	a := benchAuthorizer(b)
	b.ResetTimer()
	for b.Loop() {
		a.Enforce("u_editor", "/user/create")
	}
}

// Miss: scans all patterns (2 own + 1 grant) and all fail — worst case.
func BenchmarkEnforceMissAll(b *testing.B) {
	a := benchAuthorizer(b)
	b.ResetTimer()
	for b.Loop() {
		a.Enforce("u_editor", "/nonexistent/path")
	}
}
