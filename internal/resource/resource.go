// Package resource provides path pattern normalization and glob-based matching.
// A Resource stores a path as pre-parsed segments for zero-allocation matching.
package resource

import "strings"

// Resource holds a normalized path pattern as pre-split segments.
// Segments are stored directly (not via []byte + unsafe) for clarity.
// Wildcards: * matches one segment, ** matches zero or more and terminates the pattern.
type Resource struct {
	Segs []string // pre-parsed path segments; Match/Contains iterate directly
}

// NewResource parses a raw path string into pre-split segments, normalizing as it goes:
//   - strips leading '/'
//   - collapses duplicate slashes
//   - ** immediately terminates (everything after is discarded)
//   - * captures as a single wildcard segment
func NewResource(raw string) *Resource {
	n := len(raw)
	segs := make([]string, 0, 8)
	s := 0
	for i := range n {
		switch raw[i] {
		case '/':
			if s < i {
				segs = append(segs, raw[s:i])
			}
			s = i + 1
		case '*':
			if i+1 < n && raw[i+1] == '*' {
				if s < i {
					segs = append(segs, raw[s:i])
				}
				segs = append(segs, "**")
				return &Resource{Segs: segs}
			}
			if s < i {
				segs = append(segs, raw[s:i])
			}
			segs = append(segs, "*")
			s = i + 1
		}
	}
	if s < n {
		segs = append(segs, raw[s:n])
	}
	return &Resource{Segs: segs}
}

// String returns the normalized path, building from segments each call.
func (r *Resource) String() string {
	if len(r.Segs) == 0 {
		return "/"
	}
	return "/" + strings.Join(r.Segs, "/")
}

// Contains reports whether r subsumes r2. r contains r2 if every segment
// of r matches the corresponding segment of r2, with ** matching everything
// and * matching any single segment.
func (r *Resource) Contains(r2 *Resource) bool {
	pi, pj := 0, 0
	for pi < len(r.Segs) && pj < len(r2.Segs) {
		switch r.Segs[pi] {
		case "**":
			return true
		case "*":
			pi++
			pj++
		default:
			if r.Segs[pi] != r2.Segs[pj] {
				return false
			}
			pi++
			pj++
		}
	}
	if pi >= len(r.Segs) && pj < len(r2.Segs) {
		return false
	}
	for pi < len(r.Segs) {
		if r.Segs[pi] != "**" {
			return false
		}
		pi++
	}
	return true
}

// Match reports whether the target path matches this resource pattern.
// The target is a raw string — no pre-normalization needed. * matches one
// segment, ** matches zero or more.
func (r *Resource) Match(target string) bool {
	tp := 0
	for _, seg := range r.Segs {
		switch seg {
		case "**":
			return true
		case "*":
			if tp >= len(target) {
				return false
			}
			if target[tp] == '/' {
				tp++
			}
			if tp >= len(target) {
				return false
			}
			for tp < len(target) && target[tp] != '/' {
				tp++
			}
		default:
			if tp < len(target) && target[tp] == '/' {
				tp++
			}
			if tp >= len(target) {
				return false
			}
			te := tp
			for te < len(target) && target[te] != '/' {
				te++
			}
			if target[tp:te] != seg {
				return false
			}
			tp = te
		}
	}
	for tp < len(target) {
		if target[tp] != '/' {
			return false
		}
		tp++
	}
	return true
}
