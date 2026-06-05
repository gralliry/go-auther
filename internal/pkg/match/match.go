package match

import "strings"

const noStar = -1

// HasWildcard reports whether the pattern contains a '*' wildcard.
func HasWildcard(p string) bool {
	return strings.ContainsRune(p, '*')
}

// HasDoubleStar reports whether the pattern contains a '**' wildcard.
func HasDoubleStar(p string) bool {
	return strings.Contains(p, "**")
}

// MatchSimple matches patterns that contain only single '*' wildcards (no '**').
// No backtracking needed — a single pass through segments suffices.
func MatchSimple(p, t string) bool {
	pi, ti := 0, 0

	for {
		// Skip leading slashes.
		for pi < len(p) && p[pi] == '/' {
			pi++
		}
		for ti < len(t) && t[ti] == '/' {
			ti++
		}

		if pi >= len(p) {
			return ti >= len(t)
		}
		if ti >= len(t) {
			return pi >= len(p)
		}

		// Extract pattern segment.
		ps := pi
		for pi < len(p) && p[pi] != '/' {
			pi++
		}
		pSeg := p[ps:pi]

		// Extract target segment.
		ts := ti
		for ti < len(t) && t[ti] != '/' {
			ti++
		}
		tSeg := t[ts:ti]

		if pSeg == "*" || pSeg == tSeg {
			continue
		}
		return false
	}
}

// MatchGlob performs segment-by-segment iterative matching.
// * matches one segment, ** matches zero or more.
func MatchGlob(p, t string) bool {
	pi, ti := 0, 0
	starPi, starTi := noStar, noStar

	for {
		for pi < len(p) && p[pi] == '/' {
			pi++
		}
		for ti < len(t) && t[ti] == '/' {
			ti++
		}

		if pi >= len(p) && ti >= len(t) {
			return true
		}

		if pi >= len(p) {
			if !backtrackStar(&pi, &ti, starPi, &starTi, t) {
				return false
			}
			continue
		}

		if ti >= len(t) {
			return tailGlobStar(p, pi)
		}

		ps := pi
		for pi < len(p) && p[pi] != '/' {
			pi++
		}
		pSeg := p[ps:pi]

		if pSeg == "**" {
			starPi, starTi = pi, ti
			continue
		}

		ts := ti
		for ti < len(t) && t[ti] != '/' {
			ti++
		}
		tSeg := t[ts:ti]

		if pSeg == "*" || pSeg == tSeg {
			continue
		}

		if !backtrackStar(&pi, &ti, starPi, &starTi, t) {
			return false
		}
	}
}

// backtrackStar advances past one more target segment via the ** backtrack point.
func backtrackStar(pi, ti *int, starPi int, starTi *int, t string) bool {
	if starPi == noStar {
		return false
	}
	*pi = starPi
	*ti = *starTi
	for *ti < len(t) && t[*ti] != '/' {
		*ti++
	}
	if *ti < len(t) {
		*ti++
	}
	*starTi = *ti
	return true
}

// tailGlobStar checks whether the remaining pattern consists only of '/' and wildcards.
func tailGlobStar(p string, pi int) bool {
	for pi < len(p) {
		if p[pi] == '/' {
			pi++
			continue
		}
		if p[pi] == '*' && pi+1 < len(p) && p[pi+1] == '*' {
			pi += 2
			continue
		}
		return false
	}
	return true
}
