package resource

import "testing"

// =============================================================================
// Match tests — organized by pattern category
// =============================================================================

func TestMatch(t *testing.T) {
	// Each subtest contains a table of {pattern, target, want} cases.
	// Comments on individual cases explain counterintuitive ** greedy behavior.

	t.Run("exact", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			pattern, target string
			want            bool
		}{
			{"/user/create", "/user/create", true},
			{"/user/create", "/user/delete", false},
			{"/user/create", "/user", false},
			{"/user/create", "/user/create/extra", false},
		}
		for _, tt := range tests {
			pa := NewResource(tt.pattern)
			ta := NewResource(tt.target)
			if got := pa.Match(ta.String()); got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
			}
		}
	})

	t.Run("singleStar", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			pattern, target string
			want            bool
		}{
			{"*", "anything", true},
			{"*", "a/b", false},
			{"/*", "/anything", true},
			{"/*", "/a/b", false},
			{"/user/*", "/user/create", true},
			{"/user/*", "/user/delete", true},
			{"/user/*", "/user", false},
			{"/user/*", "/user/a/b", false},
			{"/user/*", "/user/123", true},
			{"/user/*", "/user/123/extra", false},
			{"/user/*/edit", "/user/a/edit", true},
			{"/user/*/edit", "/user/a/delete", false},
			{"/user/*/edit", "/user/a/b/edit", false},
			{"/user/*/profile/*", "/user/123/profile/edit", true},
			{"/user/*/profile/*", "/user/123/settings/edit", false},
			{"/user/*/profile/*", "/user/123/profile", false},
			{"/*/edit", "/user/edit", true},
			{"/*/edit", "/a/b/edit", false},
			{"*/b", "a/b", true},
			{"*/b", "a/c", false},
			{"*/b/c", "a/b/c", true},
			{"*/b/c", "x/b/d", false},
		}
		for _, tt := range tests {
			pa := NewResource(tt.pattern)
			ta := NewResource(tt.target)
			if got := pa.Match(ta.String()); got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
			}
		}
	})

	t.Run("doubleStar", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			pattern, target string
			want            bool
		}{
			// Basic **
			{"**", "", true},
			{"**", "a", true},
			{"**", "a/b/c", true},
			{"**", "/", true},
			{"/", "/", true},
			{"/**", "/anything", true},
			{"/**", "/a/b/c", true},
			{"/**", "/", true},
			{"/**", "/a", true},

			// ** at edges
			{"/a/**", "/a/b/c/d", true},
			{"/a/**", "/a", true},
			{"/a/**", "/b", false},
			{"/**/z", "/z", true},
			{"/**/z", "/a/b/z", true},
			{"/**/z", "/a/b/c", true}, // ** is greedy, consumes all remaining segments
			{"/api/v1/**", "/api/v1/users", true},
			{"/api/v1/**", "/api/v1/users/123", true},
			{"/api/v1/**", "/api/v2/users", false},

			// Middle **
			{"/a/**/z", "/a/z", true},
			{"/a/**/z", "/a/b/z", true},
			{"/a/**/z", "/a/b/c/z", true},
			{"/a/**/z", "/a/b/c/d", true}, // ** is greedy
			{"/a/**/z", "/x/z", false},
			{"/a/**/z", "/a/b/c/z/z", true},
			{"/data/**/export", "/data/export", true},
			{"/data/**/export", "/data/reports/export", true},
			{"/data/**/export", "/data/reports/2024/export", true},

			// Multiple **
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

			// Consecutive **
			{"/**/**", "/a", true},
			{"/**/**", "/a/b/c", true},
			{"/**/**", "/", true},
			{"/a/**/**/b", "/a/b", true},
			{"/a/**/**/b", "/a/x/b", true},
			{"/a/**/**/b", "/a/x/y/b", true},

			// Backtrack edge cases
			{"/a/**/b/**/c/d", "/a/x/y/b/z/c/d", true},
			{"/a/**/b/**/c/d", "/a/b/c/d", true},
			{"/a/**/b/**/c/**/d", "/a/b/c/d", true},
			{"/a/**/b/**/c/**/d", "/a/x/b/y/c/z/d", true},
			{"/a/**/b/**/c/**/d", "/a/x/y/z/d", true}, // ** is greedy
			{"/api/**/v2/**/data", "/api/users/v2/profiles/data", true},
			{"/api/**/v2/**/data", "/api/v2/data", true},
			{"/api/**/v2/**/data", "/api/v1/data", true}, // ** is greedy
		}
		for _, tt := range tests {
			pa := NewResource(tt.pattern)
			ta := NewResource(tt.target)
			if got := pa.Match(ta.String()); got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
			}
		}
	})

	t.Run("mixed", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			pattern, target string
			want            bool
		}{
			{"/a/*/c", "/a/b/c", true},
			{"/a/*/c", "/a/x/c", true},
			{"/a/*/c", "/a/c", false},
			{"/a/*/c", "/a/b/x/c", false},
			{"/a/*/b/**/c", "/a/x/b/c", true},
			{"/a/*/b/**/c", "/a/x/b/y/c", true},
			{"/a/*/b/**/c", "/a/x/b/y/z/c", true},
			{"/a/*/b/**/c", "/a/b/c", false},     // * needs exactly one segment before b
			{"/a/*/b/**/c", "/a/x/y/b/c", false}, // extra segment before b
			{"/a/**/b/*/c", "/a/x/b/y/c", true},
			{"/a/**/b/*/c", "/a/b/y/c", true},
			{"/a/**/b/*/c", "/a/x/y/b/z/c", true},
			{"/a/**/b/*/c", "/a/b/c", true}, // ** is greedy, * needs one segment
		}
		for _, tt := range tests {
			pa := NewResource(tt.pattern)
			ta := NewResource(tt.target)
			if got := pa.Match(ta.String()); got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
			}
		}
	})

	t.Run("edgeCases", func(t *testing.T) {
		t.Parallel()
		// Single-case sanity checks that don't fit neatly into the tables above.
		r := NewResource("/**")
		if !r.Match(NewResource("/foo").String()) {
			t.Error("/** should match any non-root path")
		}

		r2 := NewResource("**")
		if !r2.Match(NewResource("foo").String()) {
			t.Error("** should match single segment")
		}
		if !r2.Match(NewResource("foo/bar").String()) {
			t.Error("** should match multiple segments")
		}
	})
}

// =============================================================================
// Construction / normalization
// =============================================================================

func TestNew(t *testing.T) {
	t.Parallel()
	tests := []struct {
		raw  string
		want string
	}{
		{"/user/create", "/user/create"},
		{"/user//create", "/user/create"},
		{"", "/"},
		{"no-slash", "/no-slash"},
		{"/a/b", "/a/b"},
		{"/a//b", "/a/b"},
		{"/a/./b", "/a/./b"},
		{"/a/b/..", "/a/b/.."},
		{"/a/b/.", "/a/b/."},
		{"/.", "/."},
	}
	for _, tt := range tests {
		got := NewResource(tt.raw)
		if got.String() != tt.want {
			t.Errorf("New(%q) = %q, want %q", tt.raw, got.String(), tt.want)
		}
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

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
