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
		raw = "/" + raw
	}
	return Resource(path.Clean(raw))
}

// String returns the string representation of the resource.
func (r Resource) String() string { return string(r) }

// Match reports whether the target matches this resource's glob pattern.
func (r Resource) Match(target Resource) bool {
	if r == target {
		return true
	}
	s := string(r)
	if !match.HasWildcard(s) {
		return false
	}
	return match.MatchGlob(s, string(target))
}
