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
		got := HasWildcard(tt.s)
		if got != tt.want {
			t.Errorf("HasWildcard(%q) = %v, want %v", tt.s, got, tt.want)
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
		// 回溯：第一个 ** 失败后第二个 ** 接盘
		{"/a/**/x/**/z", "/a/b/c/x/y/z", true},
		{"/a/**/x/**/z", "/a/b/c/x/z", true},
		{"/a/**/x/**/z", "/a/x/y/z", true},
		{"/a/**/x/**/z", "/a/x/z", true},
	}
	for _, tt := range tests {
		got := Match(tt.pattern, tt.target)
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
		// ** 在末尾
		{"/a/**", "/a/b/c/d", true},
		{"/a/**", "/a", true},
		{"/a/**", "/b", false},
		// ** 在开头
		{"/**/z", "/z", true},
		{"/**/z", "/a/b/z", true},
		{"/**/z", "/a/b/c", false},
		// 多个 *
		{"/user/*/profile/*", "/user/123/profile/edit", true},
		{"/user/*/profile/*", "/user/123/settings/edit", false},
		{"/user/*/profile/*", "/user/123/profile", false},
		// * 在末尾
		{"/user/*", "/user/123", true},
		{"/user/*", "/user/123/extra", false},
		{"/user/*", "/user", false},
		// * 在开头
		{"/*/edit", "/user/edit", true},
		{"/*/edit", "/a/b/edit", false},
		// 纯通配符
		{"*", "anything", true},
		{"*", "a/b", false},
		{"**", "", true},
		{"**", "a", true},
		{"**", "a/b/c", true},
	}
	for _, tt := range tests {
		got := Match(tt.pattern, tt.target)
		if got != tt.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
		}
	}
}

func TestMatchBacktrackEdgeCases(t *testing.T) {
	// 这些模式考验 ** 回溯算法在复杂场景下的正确性
	tests := []struct {
		pattern string
		target  string
		want    bool
	}{
		// 第一个 ** 吃太多导致第二个 ** 无法匹配时回溯
		{"/a/**/b/**/c/d", "/a/x/y/b/z/c/d", true},
		{"/a/**/b/**/c/d", "/a/b/c/d", true},
		// 多个 ** 回溯可能多次失败
		{"/a/**/b/**/c/**/d", "/a/b/c/d", true},
		{"/a/**/b/**/c/**/d", "/a/x/b/y/c/z/d", true},
		{"/a/**/b/**/c/**/d", "/a/x/y/z/d", false},
		// ** 后跟字面匹配需要精确
		{"/a/**/z", "/a/b/c/z/z", true},   // 第二个 z 是匹配
		{"/a/**/z", "/a/b/c/z", true},
		// 长路径 + 多个通配符
		{"/api/**/v2/**/data", "/api/users/v2/profiles/data", true},
		{"/api/**/v2/**/data", "/api/v2/data", true},
		{"/api/**/v2/**/data", "/api/v1/data", false},
	}
	for _, tt := range tests {
		got := Match(tt.pattern, tt.target)
		if got != tt.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.target, got, tt.want)
		}
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
