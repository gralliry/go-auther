package match

// HasWildcard 判断字符串是否包含通配符 '*'。
func HasWildcard(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '*' {
			return true
		}
	}
	return false
}

// Match 判断目标路径是否匹配 glob 模式。
// pattern 和 target 都应为 '/' 分隔的绝对路径。
func Match(pattern, target string) bool {
	if pattern == target {
		return true
	}
	if !HasWildcard(pattern) {
		return false
	}
	pat := ParseSegs(pattern)
	tgt := ParseSegs(target)
	return DPMatch(pat, tgt)
}

// DPMatch 使用自底向上动态规划匹配模式段与目标段。
func DPMatch(pat, tgt segs) bool {
	n, m := pat.N(), tgt.N()
	rowStride := m + 1
	dp := make([]bool, (n+1)*rowStride)
	idx := func(i, j int) int { return i*rowStride + j }

	dp[idx(n, m)] = true
	for i := n - 1; i >= 0; i-- {
		if pat.At(i) == "**" {
			dp[idx(i, m)] = dp[idx(i+1, m)]
		}
	}
	for i := n - 1; i >= 0; i-- {
		pSeg := pat.At(i)
		for j := m - 1; j >= 0; j-- {
			switch {
			case pSeg == "**":
				dp[idx(i, j)] = dp[idx(i+1, j)] || dp[idx(i, j+1)]
			case pSeg == "*" || pSeg == tgt.At(j):
				dp[idx(i, j)] = dp[idx(i+1, j+1)]
			}
		}
	}
	return dp[idx(0, 0)]
}
