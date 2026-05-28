package model

import (
	"path"

	"github.com/gralliry/go-auther/internal/pkg/match"
)

// Resource represents a validated and normalized resource path pattern.
type Resource string

// NewResource validates and normalizes a resource path, returning a Resource.
func NewResource(raw string) Resource {
	if raw == "" {
		return Resource("/")
	}
	if raw[0] != '/' {
		return Resource("/" + raw)
	}
	return Resource(path.Clean(raw))
}

// String returns the string representation of the resource.
func (r Resource) String() string { return string(r) }

// Match reports whether the target matches this resource's glob pattern.
func (r Resource) Match(resource Resource) bool {
	// Exact match.
	if r == resource {
		return true
	}
	// Glob match.
	pattern := r.String()
	if !match.HasWildcard(pattern) {
		return false
	}
	// Glob match.
	return match.MatchGlob(pattern, string(resource))
}
