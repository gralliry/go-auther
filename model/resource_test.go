package model

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
		got := Resource(tt.pattern).Match(tt.target)
		if got != tt.want {
			t.Errorf("Resource(%q).Match(%q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
		}
	}
}

func TestMatchRootOnly(t *testing.T) {
	if !Resource("/**").Match("/foo") {
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
		got := Resource(tt.pattern).Match(tt.target)
		if got != tt.want {
			t.Errorf("Resource(%q).Match(%q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
		}
	}
}

func TestMatchDoubleStarAlone(t *testing.T) {
	if !Resource("**").Match("foo") {
		t.Error("** should match single segment")
	}
	if !Resource("**").Match("foo/bar") {
		t.Error("** should match multiple segments")
	}
}

func TestHasWildcard(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"", false},
		{"/", false},
		{"/user/create", false},
		{"/user/*", true},
		{"/**", true},
		{"*", true},
		{"**", true},
		{"/a/*/b/**/c", true},
		{"no wildcards here", false},
	}
	for _, tt := range tests {
		got := Resource(tt.s).HasWildcard()
		if got != tt.want {
			t.Errorf("Resource(%q).HasWildcard() = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestMatchMultipleDoubleStars(t *testing.T) {
	tests := []struct {
		pattern string
		target  string
		want    bool
	}{
		{"/a/**/b/**/c", "/a/x/y/b/z/c", true},
		{"/a/**/b/**/c", "/a/b/c", true},
		{"/a/**/b/**/c", "/a/b/x/c", true},
		{"/a/**/b/**/c", "/a/x/b/c", true},
		{"/a/**/b/**/c", "/a/c", false},
		{"/a/**/b/**/c", "/x/b/c", false},
		{"/a/**/x/**/z", "/a/b/c/x/y/z", true},
		{"/a/**/x/**/z", "/a/b/c/x/z", true},
		{"/a/**/x/**/z", "/a/x/y/z", true},
		{"/a/**/x/**/z", "/a/x/z", true},
	}
	for _, tt := range tests {
		got := Resource(tt.pattern).Match(tt.target)
		if got != tt.want {
			t.Errorf("Resource(%q).Match(%q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
		}
	}
}

func TestMatchDoubleStarAtEdges(t *testing.T) {
	tests := []struct {
		pattern string
		target  string
		want    bool
	}{
		{"/a/**", "/a/b/c/d", true},
		{"/a/**", "/a", true},
		{"/a/**", "/b", false},
		{"/**/z", "/z", true},
		{"/**/z", "/a/b/z", true},
		{"/**/z", "/a/b/c", false},
		{"/user/*/profile/*", "/user/123/profile/edit", true},
		{"/user/*/profile/*", "/user/123/settings/edit", false},
		{"/user/*/profile/*", "/user/123/profile", false},
		{"/user/*", "/user/123", true},
		{"/user/*", "/user/123/extra", false},
		{"/user/*", "/user", false},
		{"/*/edit", "/user/edit", true},
		{"/*/edit", "/a/b/edit", false},
		{"*", "anything", true},
		{"*", "a/b", false},
		{"**", "", true},
		{"**", "a", true},
		{"**", "a/b/c", true},
	}
	for _, tt := range tests {
		got := Resource(tt.pattern).Match(tt.target)
		if got != tt.want {
			t.Errorf("Resource(%q).Match(%q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
		}
	}
}

func TestMatchBacktrackEdgeCases(t *testing.T) {
	tests := []struct {
		pattern string
		target  string
		want    bool
	}{
		{"/a/**/b/**/c/d", "/a/x/y/b/z/c/d", true},
		{"/a/**/b/**/c/d", "/a/b/c/d", true},
		{"/a/**/b/**/c/**/d", "/a/b/c/d", true},
		{"/a/**/b/**/c/**/d", "/a/x/b/y/c/z/d", true},
		{"/a/**/b/**/c/**/d", "/a/x/y/z/d", false},
		{"/a/**/z", "/a/b/c/z/z", true},
		{"/a/**/z", "/a/b/c/z", true},
		{"/api/**/v2/**/data", "/api/users/v2/profiles/data", true},
		{"/api/**/v2/**/data", "/api/v2/data", true},
		{"/api/**/v2/**/data", "/api/v1/data", false},
	}
	for _, tt := range tests {
		got := Resource(tt.pattern).Match(tt.target)
		if got != tt.want {
			t.Errorf("Resource(%q).Match(%q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
		}
	}
}

func TestNewResource(t *testing.T) {
	tests := []struct {
		raw     string
		want    string
		wantErr bool
	}{
		{"/user/create", "/user/create", false},
		{"/user//create", "/user/create", false},
		{"/user/./create", "/user/create", false},
		{"/user/../create", "/create", false},
		{"/trailing/", "/trailing", false},
		{"", "", true},
		{"no-slash", "", true},
	}
	for _, tt := range tests {
		got, err := NewResource(tt.raw)
		if tt.wantErr {
			if err == nil {
				t.Errorf("NewResource(%q) should error", tt.raw)
			}
			continue
		}
		if err != nil {
			t.Errorf("NewResource(%q): %v", tt.raw, err)
			continue
		}
		if string(got) != tt.want {
			t.Errorf("NewResource(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

var benchResult bool

func BenchmarkMatchExact(b *testing.B) {
	pat := Resource("/user/create")
	for i := 0; i < b.N; i++ {
		benchResult = pat.Match("/user/create")
	}
}

func BenchmarkMatchSingleStar(b *testing.B) {
	pat := Resource("/user/*/edit")
	for i := 0; i < b.N; i++ {
		benchResult = pat.Match("/user/alice/edit")
	}
}

func BenchmarkMatchDoubleStar(b *testing.B) {
	pat := Resource("/a/**/z")
	for i := 0; i < b.N; i++ {
		benchResult = pat.Match("/a/b/c/d/e/z")
	}
}

func BenchmarkMatchNoMatch(b *testing.B) {
	pat := Resource("/user/create")
	for i := 0; i < b.N; i++ {
		benchResult = pat.Match("/user/delete")
	}
}

func BenchmarkMatchLongDoubleStar(b *testing.B) {
	pat := Resource("/api/v1/**")
	for i := 0; i < b.N; i++ {
		benchResult = pat.Match("/api/v1/users/admin/permissions/read/write")
	}
}
