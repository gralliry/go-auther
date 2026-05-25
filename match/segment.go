// Package match 提供资源路径的 glob 匹配引擎。
// 支持 *（单段匹配）和 **（零或多段匹配）通配符。
package match

// segs 保存路径字符串中各个路径段的起止位置。
type segs struct {
	s  string // 原始路径字符串
	p  []int  // 各段起始位置
	on []int  // 各段结束位置（不包含）
}

// ParseSegs 将路径解析为路径段视图，不产生额外的字符串分配。
func ParseSegs(path string) segs {
	if path == "/" {
		return segs{s: "/", p: []int{0}, on: []int{1}}
	}
	n := 1
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			n++
		}
	}
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

// N 返回路径段的数量。
func (sg segs) N() int { return len(sg.p) }

// At 返回第 i 个路径段的字符串。
func (sg segs) At(i int) string {
	if i >= len(sg.p) {
		return ""
	}
	return sg.s[sg.p[i]:sg.on[i]]
}
