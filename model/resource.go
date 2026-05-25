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

// Match 判断 target 是否匹配此资源模式，零堆分配。
func (r Resource) Match(target string) bool {
	p := string(r)
	if p == target {
		return true
	}
	if !r.HasWildcard() {
		return false
	}

	pi, ti := 0, 0
	starPi, starTi := noStar, noStar

	for {
		for pi < len(p) && p[pi] == '/' {
			pi++
		}
		for ti < len(target) && target[ti] == '/' {
			ti++
		}

		if pi >= len(p) && ti >= len(target) {
			return true
		}

		if pi >= len(p) {
			if !backtrackStar(&pi, &ti, starPi, &starTi, target) {
				return false
			}
			continue
		}

		if ti >= len(target) {
			// 目标耗尽，剩余模式只能包含 '/' 和 **。
			for pi < len(p) {
				if p[pi] == '/' {
					pi++
				} else if p[pi] == '*' && pi+1 < len(p) && p[pi+1] == '*' {
					pi += 2
				} else {
					return false
				}
			}
			return true
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
		for ti < len(target) && target[ti] != '/' {
			ti++
		}
		tSeg := target[ts:ti]

		if pSeg == "*" || pSeg == tSeg {
			continue
		}

		if !backtrackStar(&pi, &ti, starPi, &starTi, target) {
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
