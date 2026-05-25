package model

import (
	"fmt"
	"path"
)

const noStar = -1

// Resource 是一个经过校验和规范化的资源路径模式。
// 支持 *（匹配单个路径段）和 **（匹配零个或多个路径段）通配符。
type Resource string

// NewResource 校验并规范化一个原始资源路径。
func NewResource(raw string) (Resource, error) {
	if raw == "" {
		return "", fmt.Errorf("resource must not be empty")
	}
	if raw[0] != '/' {
		return "", fmt.Errorf("resource must start with '/'")
	}
	return Resource(path.Clean(raw)), nil
}

// Match 判断 target 是否匹配此资源模式。
func (r Resource) Match(target string) bool {
	if string(r) == target {
		return true
	}
	if !r.HasWildcard() {
		return false
	}
	return matchGlob(string(r), target)
}

// HasWildcard 判断此模式是否包含 '*' 通配符。
func (r Resource) HasWildcard() bool {
	for i := 0; i < len(r); i++ {
		if r[i] == '*' {
			return true
		}
	}
	return false
}

// String 返回规范化后的路径字符串。
func (r Resource) String() string { return string(r) }

// matchGlob 分段迭代匹配，零堆分配。
// * 匹配单段，** 匹配零或多段。
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
