package match

import "testing"

func TestMatchExact(t *testing.T) {
	tests := []struct {
		pattern string
		target  string
		want    bool
	}{
		{"/**", "/anything", true},
		{"/**", "/a/b/c", true},
		{"/**", "/", true},
		{"/user/create", "/user/create", true},
		{"/user/create", "/user/delete", false},
		{"/user/create", "/user", false},
		{"/user/create", "/user/create/extra", false},
		{"/user/*", "/user/create", true},
		{"/user/*", "/user/delete", true},
		{"/user/*", "/user", false},
		{"/user/*", "/user/a/b", false},
		{"/user/*/edit", "/user/a/edit", true},
		{"/user/*/edit", "/user/a/delete", false},
		{"/user/*/edit", "/user/a/b/edit", false},
		{"/a/**/z", "/a/z", true},
		{"/a/**/z", "/a/b/z", true},
		{"/a/**/z", "/a/b/c/z", true},
		{"/a/**/z", "/a/b/c/d", false},
		{"/a/**/z", "/x/z", false},
		{"/a/*/c", "/a/b/c", true},
		{"/a/*/c", "/a/x/c", true},
		{"/a/*/c", "/a/c", false},
		{"/a/*/c", "/a/b/x/c", false},
		{"/api/v1/**", "/api/v1/users", true},
		{"/api/v1/**", "/api/v1/users/123", true},
		{"/api/v1/**", "/api/v2/users", false},
		{"/data/**/export", "/data/export", true},
		{"/data/**/export", "/data/reports/export", true},
		{"/data/**/export", "/data/reports/2024/export", true},
	}
	for _, tt := range tests {
		got := Match(tt.pattern, tt.target)
		if got != tt.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
		}
	}
}

func TestMatchRootOnly(t *testing.T) {
	if !Match("/**", "/foo") {
		t.Error("/** should match any non-root path")
	}
}

func TestMatchEdgeCases(t *testing.T) {
	tests := []struct {
		pattern string
		target  string
		want    bool
	}{
		{"/*", "/anything", true},
		{"/*", "/a/b", false},
		{"/", "/", true},
	}
	for _, tt := range tests {
		got := Match(tt.pattern, tt.target)
		if got != tt.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
		}
	}
}

func TestMatchDoubleStarAlone(t *testing.T) {
	// ** alone should match zero or more segments
	if !Match("**", "foo") {
		t.Error("** should match single segment")
	}
	if !Match("**", "foo/bar") {
		t.Error("** should match multiple segments")
	}
}

// Benchmarks

var benchResult bool

func BenchmarkMatchExact(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchResult = Match("/user/create", "/user/create")
	}
}

func BenchmarkMatchSingleStar(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchResult = Match("/user/*/edit", "/user/alice/edit")
	}
}

func BenchmarkMatchDoubleStar(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchResult = Match("/a/**/z", "/a/b/c/d/e/z")
	}
}

func BenchmarkMatchNoMatch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchResult = Match("/user/create", "/user/delete")
	}
}

func BenchmarkMatchLongDoubleStar(b *testing.B) {
	for i := 0; i < b.N; i++ {
		benchResult = Match("/api/v1/**", "/api/v1/users/admin/permissions/read/write")
	}
}
