package resource

import (
	"testing"
)

// =============================================================================
// Construction / normalization tests
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
