package resource

import (
	"fmt"
	"path"
)

// Resource represents a validated and normalized resource path pattern.
type Resource string

// New validates and normalizes a resource path, returning a Resource.
func New(raw string) (Resource, error) {
	if raw == "" {
		return "", fmt.Errorf("resource must not be empty")
	}
	if raw[0] != '/' {
		return "", fmt.Errorf("resource must start with '/'")
	}
	return Resource(path.Clean(raw)), nil
}

// String returns the string representation of the resource.
func (r Resource) String() string { return string(r) }

// Match reports whether the target matches this resource's glob pattern.
func (r Resource) Match(target string) bool {
	pattern := r.String()
	if pattern == target {
		return true
	}
	if !r.hasWildcard() {
		return false
	}
	return r.matchGlob(pattern, target)
}
