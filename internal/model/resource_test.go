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
		{"/a/**/z", "/a/b/c/d", true}, // ** is greedy, consumes all remaining segments
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
		pa := NewResource(tt.pattern)
		ta := NewResource(tt.target)
		got := pa.Match(ta.String())
		if got != tt.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
		}
	}
}

func TestMatchRootOnly(t *testing.T) {
	p := NewResource("/**")
	ta := NewResource("/foo")
	if !p.Match(ta.String()) {
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
		pa := NewResource(tt.pattern)
		ta := NewResource(tt.target)
		got := pa.Match(ta.String())
		if got != tt.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
		}
	}
}

func TestMatchDoubleStarAlone(t *testing.T) {
	p := NewResource("**")
	ta := NewResource("foo")
	if !p.Match(ta.String()) {
		t.Error("** should match single segment")
	}
	ta2 := NewResource("foo/bar")
	if !p.Match(ta2.String()) {
		t.Error("** should match multiple segments")
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
		{"/a/**/b/**/c", "/a/c", true}, // ** is greedy
		{"/a/**/b/**/c", "/x/b/c", false},
		{"/a/**/x/**/z", "/a/b/c/x/y/z", true},
		{"/a/**/x/**/z", "/a/b/c/x/z", true},
		{"/a/**/x/**/z", "/a/x/y/z", true},
		{"/a/**/x/**/z", "/a/x/z", true},
	}
	for _, tt := range tests {
		pa := NewResource(tt.pattern)
		ta := NewResource(tt.target)
		got := pa.Match(ta.String())
		if got != tt.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
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
		{"/**/z", "/a/b/c", true}, // ** is greedy
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
		pa := NewResource(tt.pattern)
		ta := NewResource(tt.target)
		got := pa.Match(ta.String())
		if got != tt.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
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
		{"/a/**/b/**/c/**/d", "/a/x/y/z/d", true}, // ** is greedy
		{"/a/**/z", "/a/b/c/z/z", true},
		{"/a/**/z", "/a/b/c/z", true},
		{"/api/**/v2/**/data", "/api/users/v2/profiles/data", true},
		{"/api/**/v2/**/data", "/api/v2/data", true},
		{"/api/**/v2/**/data", "/api/v1/data", true}, // ** is greedy
	}
	for _, tt := range tests {
		pa := NewResource(tt.pattern)
		ta := NewResource(tt.target)
		got := pa.Match(ta.String())
		if got != tt.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
		}
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"/user/create", "/user/create"},
		{"/user//create", "/user/create"},
		{"", "/"},
		{"no-slash", "/no-slash"},
	}
	for _, tt := range tests {
		got := NewResource(tt.raw)
		if got.String() != tt.want {
			t.Errorf("New(%q) = %q, want %q", tt.raw, got.String(), tt.want)
		}
	}
}

var benchResult bool

func BenchmarkMatchExact(b *testing.B) {
	r := NewResource("/user/create")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := NewResource("/user/create")
		benchResult = r.Match(t.String())
	}
}

func BenchmarkMatchSingleStar(b *testing.B) {
	r := NewResource("/user/*/edit")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := NewResource("/user/alice/edit")
		benchResult = r.Match(t.String())
	}
}

func BenchmarkMatchDoubleStar(b *testing.B) {
	r := NewResource("/a/**/z")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := NewResource("/a/b/c/d/e/z")
		benchResult = r.Match(t.String())
	}
}

func BenchmarkMatchNoMatch(b *testing.B) {
	r := NewResource("/user/create")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := NewResource("/user/delete")
		benchResult = r.Match(t.String())
	}
}

func BenchmarkMatchLongDoubleStar(b *testing.B) {
	r := NewResource("/api/v1/**")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := NewResource("/api/v1/users/admin/permissions/read/write")
		benchResult = r.Match(t.String())
	}
}

func TestMatchStarAtStart(t *testing.T) {
	tests := []struct {
		pattern string
		target  string
		want    bool
	}{
		{"*", "anything", true},
		{"*", "a/b", false},
		{"*/b", "a/b", true},
		{"*/b", "a/c", false},
		{"*/b/c", "a/b/c", true},
		{"*/b/c", "x/b/d", false},
	}
	for _, tt := range tests {
		pa := NewResource(tt.pattern)
		ta := NewResource(tt.target)
		got := pa.Match(ta.String())
		if got != tt.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
		}
	}
}

func TestMatchComplexMixedGlobs(t *testing.T) {
	tests := []struct {
		pattern string
		target  string
		want    bool
	}{
		{"/a/*/b/**/c", "/a/x/b/c", true},
		{"/a/*/b/**/c", "/a/x/b/y/c", true},
		{"/a/*/b/**/c", "/a/x/b/y/z/c", true},
		{"/a/*/b/**/c", "/a/b/c", false},     // * needs exactly one segment
		{"/a/*/b/**/c", "/a/x/y/b/c", false}, // extra segment before b
		{"/a/**/b/*/c", "/a/x/b/y/c", true},
		{"/a/**/b/*/c", "/a/b/y/c", true},
		{"/a/**/b/*/c", "/a/x/y/b/z/c", true},
		{"/a/**/b/*/c", "/a/b/c", true}, // ** is greedy // * needs exactly one segment
	}
	for _, tt := range tests {
		pa := NewResource(tt.pattern)
		ta := NewResource(tt.target)
		got := pa.Match(ta.String())
		if got != tt.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
		}
	}
}

func TestMatchDoubleStarRecursive(t *testing.T) {
	tests := []struct {
		pattern string
		target  string
		want    bool
	}{
		{"/**/**", "/a", true},
		{"/**/**", "/a/b/c", true},
		{"/**/**", "/", true},
		{"/a/**/**/b", "/a/b", true},
		{"/a/**/**/b", "/a/x/b", true},
		{"/a/**/**/b", "/a/x/y/b", true},
	}
	for _, tt := range tests {
		pa := NewResource(tt.pattern)
		ta := NewResource(tt.target)
		got := pa.Match(ta.String())
		if got != tt.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
		}
	}
}

func TestMatchOnlyDoubleStar(t *testing.T) {
	tests := []struct {
		pattern string
		target  string
		want    bool
	}{
		{"**", "", true},
		{"**", "/", true},
		{"**", "a", true},
		{"**", "a/b/c/d/e", true},
		{"/**", "/", true},
		{"/**", "/a", true},
		{"/**", "/a/b/c", true},
	}
	for _, tt := range tests {
		pa := NewResource(tt.pattern)
		ta := NewResource(tt.target)
		got := pa.Match(ta.String())
		if got != tt.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
		}
	}
}

func TestNewNormalization(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"/a/b", "/a/b"},
		{"/a//b", "/a/b"},
		{"/a/./b", "/a/./b"},
		{"/a/b/..", "/a/b/.."},
		{"/a/b/.", "/a/b/."},
		{"/.", "/."},
	}
	for _, tt := range tests {
		r := NewResource(tt.raw)
		if r.String() != tt.want {
			t.Errorf("New(%q) = %q, want %q", tt.raw, r.String(), tt.want)
		}
	}
}
