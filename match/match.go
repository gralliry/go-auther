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
	return matchIter(pattern, target)
}

// matchIter 使用分段迭代 + 回溯实现 glob 匹配，零堆分配。
// * 匹配单段（不跨越 '/'），** 匹配零或多段。
func matchIter(p, t string) bool {
	pi, ti := 0, 0 // 字符级指针
	starPS, starTS := -1, -1 // ** 回溯点：模式段结束位置, 目标位置

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

		// 模式耗尽但目标未耗尽 → 回溯或失败
		if pi >= len(p) {
			if starPS != -1 {
				pi = starPS
				ti = starTS
				// ** 多消耗一个目标段（含尾部 '/'）
				for ti < len(t) && t[ti] != '/' {
					ti++
				}
				if ti < len(t) && t[ti] == '/' {
					ti++
				}
				starTS = ti
				continue
			}
			return false
		}

		// 目标耗尽，剩余模式只能包含 **
		if ti >= len(t) {
			for pi < len(p) {
				if p[pi] == '/' {
					pi++
					continue
				}
				if p[pi] == '*' && pi+1 < len(p) && p[pi+1] == '*' {
					pi += 2
					continue
				}
				// * 需要消费一段，目标已耗尽 → 失败
				return false
			}
			return true
		}

		// 提取当前模式段
		pStart := pi
		for pi < len(p) && p[pi] != '/' {
			pi++
		}
		pSeg := p[pStart:pi]

		// **：零或多段
		if pSeg == "**" {
			starPS = pi // 下一模式段起始位置
			starTS = ti // 当前目标段起始位置
			continue
		}

		// 提取当前目标段
		tStart := ti
		for ti < len(t) && t[ti] != '/' {
			ti++
		}
		tSeg := t[tStart:ti]

		// * 或字面匹配
		if pSeg == "*" || pSeg == tSeg {
			continue
		}

		// 不匹配，尝试 ** 回溯
		if starPS != -1 {
			pi = starPS
			ti = starTS
			// ** 多消耗一个目标段
			for ti < len(t) && t[ti] != '/' {
				ti++
			}
			if ti < len(t) && t[ti] == '/' {
				ti++
			}
			starTS = ti
			continue
		}

		return false
	}
}
