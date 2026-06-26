package resource

import "testing"

var benchBool bool

// =============================================================================
// NewResource + Match — pattern created each iteration, raw string target
// =============================================================================

func BenchmarkNewMatchExact(b *testing.B) {
	for b.Loop() {
		r := NewResource("/user/create")
		benchBool = r.Match("/user/create")
	}
}

func BenchmarkNewMatchWildcard(b *testing.B) {
	for b.Loop() {
		r := NewResource("/user/*/edit")
		benchBool = r.Match("/user/alice/edit")
	}
}

func BenchmarkNewMatchDoubleStar(b *testing.B) {
	for b.Loop() {
		r := NewResource("/api/v1/**")
		benchBool = r.Match("/api/v1/users/admin/data")
	}
}

// =============================================================================
// Match via raw string — pattern pre-created, raw target per iteration
// =============================================================================

func BenchmarkMatchExact(b *testing.B) {
	r := NewResource("/user/create")
	for b.Loop() {
		benchBool = r.Match("/user/create")
	}
}

func BenchmarkMatchWildcard(b *testing.B) {
	r := NewResource("/user/*/edit")
	for b.Loop() {
		benchBool = r.Match("/user/alice/edit")
	}
}

func BenchmarkMatchDoubleStar(b *testing.B) {
	r := NewResource("/api/v1/**")
	for b.Loop() {
		benchBool = r.Match("/api/v1/users/admin/data")
	}
}

func BenchmarkMatchNoMatch(b *testing.B) {
	r := NewResource("/user/create")
	for b.Loop() {
		benchBool = r.Match("/user/delete")
	}
}

func BenchmarkMatchLongPath(b *testing.B) {
	r := NewResource("/api/v1/**")
	for b.Loop() {
		benchBool = r.Match("/api/v1/users/admin/permissions/read/write")
	}
}

// =============================================================================
// Match edge cases — unnormalized input: no leading /, double slashes, etc.
// =============================================================================

func BenchmarkMatchNoLeadingSlash(b *testing.B) {
	r := NewResource("/user/create")
	for b.Loop() {
		benchBool = r.Match("user/create")
	}
}

func BenchmarkMatchDoubleSlash(b *testing.B) {
	r := NewResource("/user/*/edit")
	for b.Loop() {
		benchBool = r.Match("//user//alice/edit")
	}
}

func BenchmarkMatchTrailingSlash(b *testing.B) {
	r := NewResource("/user/create")
	for b.Loop() {
		benchBool = r.Match("/user/create/")
	}
}
