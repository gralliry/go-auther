package match

import (
	"fmt"
	"path"
)

const noStar = -1

// Clean 校验并规范化一个资源路径字符串。
func Clean(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("resource must not be empty")
	}
	if raw[0] != '/' {
		return "", fmt.Errorf("resource must start with '/'")
	}
	return path.Clean(raw), nil
}

// Match 判断 target 是否匹配 pattern glob 模式，零堆分配。
func Match(pattern, target string) bool {
	if pattern == target {
		return true
	}
	if !HasWildcard(pattern) {
		return false
	}
	return matchGlob(pattern, target)
}

// HasWildcard 判断字符串是否包含 '*' 通配符。
func HasWildcard(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '*' {
			return true
		}
	}
	return false
}

// matchGlob 分段迭代匹配。* 匹配单段，** 匹配零或多段。
func matchGlob(p, t string) bool {
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
		*ti++
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
