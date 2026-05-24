package auther

// match reports whether target matches the glob pattern.
// Both pattern and target should be '/' delimited absolute paths.
//
// Supported wildcards:
//   - '*'  matches exactly one path segment (does not cross '/')
//   - '**' matches zero or more path segments
func match(pattern, target string) bool {
	// Fast path: exact match.
	if pattern == target {
		return true
	}

	// Fast path: no wildcards means literal comparison already failed.
	if !hasWildcard(pattern) {
		return false
	}

	// Parse pattern and target into segment views (start offsets).
	// This avoids string allocation from strings.Split.
	pat := parseSegs(pattern)
	tgt := parseSegs(target)
	return matchDP(pat, tgt)
}

// =============================================================================
// Segment parsing
// =============================================================================

// segs holds segment start positions for a path string.
// segs[i] is the byte offset where segment i begins.
type segs struct {
	s  string  // the original path
	p  []int   // segment start positions (lazily filled indices)
	on []int   // .. offsets for each path char (lazily built)
}

// parseSegs builds a segs view for a path.
func parseSegs(path string) segs {
	if path == "/" {
		return segs{s: "/", p: []int{0}}
	}
	// First pass: count segments.
	n := 1
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			n++
		}
	}
	// Second pass: record start positions.
	// Skip the leading empty segment from the leading '/'.
	p := make([]int, 0, n)
	for i := 0; i < len(path); {
		if path[i] == '/' {
			i++
			continue
		}
		p = append(p, i)
		for i < len(path) && path[i] != '/' {
			i++
		}
	}
	return segs{s: path, p: p}
}

// n returns the number of segments.
func (sg segs) n() int { return len(sg.p) }

// at returns the i-th segment as a string (without allocation when cached).
func (sg segs) at(i int) string {
	if i >= len(sg.p) {
		return ""
	}
	start := sg.p[i]
	end := start
	for end < len(sg.s) && sg.s[end] != '/' {
		end++
	}
	return sg.s[start:end]
}

// eq returns true if segment i equals literal.
func (sg segs) eq(i int, literal string) bool {
	return sg.at(i) == literal
}

// =============================================================================
// DP matching
// =============================================================================

// matchDP performs bottom-up DP matching of pattern segments against target
// segments. The DP table uses a flat slice for better cache locality.
// dp(i, j) = does pat[i:] match tgt[j:]?
func matchDP(pat, tgt segs) bool {
	n, m := pat.n(), tgt.n()

	// dp[i][j] stored at index i*(m+1) + j.
	rowStride := m + 1
	dp := make([]bool, (n+1)*rowStride)
	idx := func(i, j int) int { return i*rowStride + j }

	// Base: both empty → true.
	dp[idx(n, m)] = true

	// Base: when target is empty, only patterns of all-** can match.
	for i := n - 1; i >= 0; i-- {
		if pat.at(i) == "**" {
			dp[idx(i, m)] = dp[idx(i+1, m)]
		}
		// else stays false
	}

	// Fill table bottom-up.
	for i := n - 1; i >= 0; i-- {
		pSeg := pat.at(i)
		for j := m - 1; j >= 0; j-- {
			switch {
			case pSeg == "**":
				// ** matches zero segments (skip it) OR one/more (skip target segment).
				dp[idx(i, j)] = dp[idx(i+1, j)] || dp[idx(i, j+1)]
			case pSeg == "*" || pSeg == tgt.at(j):
				// * or literal match consumes one segment from both.
				dp[idx(i, j)] = dp[idx(i+1, j+1)]
			}
			// else: mismatch → stays false
		}
	}
	return dp[idx(0, 0)]
}

// =============================================================================
// Helpers
// =============================================================================

func hasWildcard(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '*' {
			return true
		}
	}
	return false
}
