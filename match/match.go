package match

const noStar = -1

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
func Match(pattern, target string) bool {
	if pattern == target {
		return true
	}
	if !HasWildcard(pattern) {
		return false
	}
	return matchGlob(pattern, target)
}

// matchGlob 分段迭代匹配，零堆分配。
// * 匹配单段，** 匹配零或多段。
func matchGlob(p, t string) bool {
	pi, ti := 0, 0
	starPi, starTi := noStar, noStar

	for {
		// 跳过前导 '/'
		for pi < len(p) && p[pi] == '/' {
			pi++
		}
		for ti < len(t) && t[ti] == '/' {
			ti++
		}

		// 两者均耗尽 → 匹配成功
		if pi >= len(p) && ti >= len(t) {
			return true
		}

		// 模式耗尽，尝试 ** 回溯多消耗一段
		if pi >= len(p) {
			if !backtrackStar(&pi, &ti, starPi, &starTi, t) {
				return false
			}
			continue
		}

		// 目标耗尽，剩余模式只能包含 ** 和 '/'
		if ti >= len(t) {
			return tailGlobStar(p, pi)
		}

		// 提取当前段（到下一个 '/' 为止）
		ps := pi
		for pi < len(p) && p[pi] != '/' {
			pi++
		}
		pSeg := p[ps:pi]

		// **：零或多段，记录回溯点
		if pSeg == "**" {
			starPi, starTi = pi, ti
			continue
		}

		// 提取目标段
		ts := ti
		for ti < len(t) && t[ti] != '/' {
			ti++
		}
		tSeg := t[ts:ti]

		// 字面匹配或 * 单段匹配
		if pSeg == "*" || pSeg == tSeg {
			continue
		}

		// 不匹配，尝试 ** 回溯
		if !backtrackStar(&pi, &ti, starPi, &starTi, t) {
			return false
		}
	}
}

// backtrackStar 尝试通过 ** 回溯点多消耗目标的一段。
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
		*ti++ // skip '/'
	}
	*starTi = *ti
	return true
}

// tailGlobStar 检查 pattern 剩余部分是否只包含 '/' 和通配符。
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
