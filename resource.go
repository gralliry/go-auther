package auther

import "fmt"

// match 判断目标路径是否匹配 glob 模式。
// pattern 和 target 都应为 '/' 分隔的绝对路径。
//
// 支持的通配符：
//   - '*'  匹配恰好一个路径段（不跨越 '/'）
//   - '**' 匹配零个或多个路径段
func match(pattern, target string) bool {
	// 快速路径：完全一致直接返回。
	if pattern == target {
		return true
	}

	// 快速路径：没有通配符时，字面量比较失败即直接返回。
	if !hasWildcard(pattern) {
		return false
	}

	// 将 pattern 和 target 解析为路径段视图（起始偏移量），
	// 避免 strings.Split 带来的字符串分配开销。
	pat := parseSegs(pattern)
	tgt := parseSegs(target)
	return matchDP(pat, tgt)
}

// segs 保存路径字符串中各个路径段的起止位置。
type segs struct {
	s  string // 原始路径字符串
	p  []int  // 各段起始位置
	on []int  // 各段结束位置（不包含）
}

// parseSegs 将路径解析为 segs 结构，不产生额外的字符串分配。
func parseSegs(path string) segs {
	if path == "/" {
		return segs{s: "/", p: []int{0}, on: []int{1}}
	}
	// 第一趟：统计路径段数量。
	n := 1
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			n++
		}
	}
	// 第二趟：记录每段的起止位置，跳过路径开头的 '/'。
	p := make([]int, 0, n)
	on := make([]int, 0, n)
	for i := 0; i < len(path); {
		if path[i] == '/' {
			i++
			continue
		}
		p = append(p, i)
		for i < len(path) && path[i] != '/' {
			i++
		}
		on = append(on, i)
	}
	return segs{s: path, p: p, on: on}
}

// n 返回路径段的数量。
func (sg segs) n() int { return len(sg.p) }

// at 返回第 i 个路径段的字符串。
func (sg segs) at(i int) string {
	if i >= len(sg.p) {
		return ""
	}
	return sg.s[sg.p[i]:sg.on[i]]
}

// eq 判断第 i 个路径段是否等于指定字符串。
func (sg segs) eq(i int, literal string) bool {
	return sg.at(i) == literal
}

// matchDP 使用自底向上的动态规划算法匹配模式段与目标段。
// DP 表使用扁平切片存储以提高 CPU 缓存局部性。
// dp(i, j) 表示 pat[i:] 是否匹配 tgt[j:]。
func matchDP(pat, tgt segs) bool {
	n, m := pat.n(), tgt.n()

	// dp[i][j] 存储在 index i*(m+1) + j 的位置。
	rowStride := m + 1
	dp := make([]bool, (n+1)*rowStride)
	idx := func(i, j int) int { return i*rowStride + j }

	// 基本情况：两者都为空 → 匹配成功。
	dp[idx(n, m)] = true

	// 基本情况：目标为空时，只有全部由 ** 组成的模式才能匹配。
	for i := n - 1; i >= 0; i-- {
		if pat.at(i) == "**" {
			dp[idx(i, m)] = dp[idx(i+1, m)]
		}
	}

	// 自底向上填充 DP 表。
	for i := n - 1; i >= 0; i-- {
		pSeg := pat.at(i)
		for j := m - 1; j >= 0; j-- {
			switch {
			case pSeg == "**":
				// ** 匹配零段（跳过模式段）或一段及以上（跳过目标段）。
				dp[idx(i, j)] = dp[idx(i+1, j)] || dp[idx(i, j+1)]
			case pSeg == "*" || pSeg == tgt.at(j):
				// * 或字面量匹配 → 各消耗一段。
				dp[idx(i, j)] = dp[idx(i+1, j+1)]
			}
		}
	}
	return dp[idx(0, 0)]
}

// normalizeResource 校验并规范化资源路径：
//   - 不能为空
//   - 必须以 '/' 开头
//   - 合并连续的斜杠（// → /）
//   - 去除尾部斜杠（根路径 "/" 除外）
func normalizeResource(resource string) (string, error) {
	if resource == "" {
		return "", fmt.Errorf("%w: resource must not be empty", ErrInvalidResource)
	}
	if resource[0] != '/' {
		return "", fmt.Errorf("%w: resource must start with '/'", ErrInvalidResource)
	}

	result := make([]byte, 0, len(resource))
	result = append(result, '/')
	for i := 1; i < len(resource); i++ {
		if resource[i] == '/' && resource[i-1] == '/' {
			continue
		}
		result = append(result, resource[i])
	}

	if len(result) > 1 && result[len(result)-1] == '/' {
		result = result[:len(result)-1]
	}

	return string(result), nil
}

// hasWildcard 判断字符串是否包含通配符 '*'。
func hasWildcard(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '*' {
			return true
		}
	}
	return false
}
