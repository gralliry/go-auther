package model

// Resource represents a normalized resource pattern.
type Resource struct {
	raw string
}

func NewResource(raw string) *Resource {
	stack := []byte{'/'}
	for i := 0; i < len(raw); i++ {
		cur := raw[i]
		lst := stack[len(stack)-1]
		// 去除重复的 '/'
		switch lst {
		case '/':
			switch cur {
			case '/':
				continue
			case '*':
				stack = append(stack, '*')
			default:
				stack = append(stack, cur)
			}
		case '*':
			switch cur {
			case '/':
				stack = append(stack, '/')
			case '*':
				stack = append(stack, '*')
				goto end
			default:
				stack = append(stack, '/', cur)
			}
		default:
			switch cur {
			case '/':
				stack = append(stack, '/')
			case '*':
				stack = append(stack, '/', '*')
			default:
				stack = append(stack, cur)
			}
		}
	}
end:
	// 移除尾部的 '/'
	if len(stack) > 1 && stack[len(stack)-1] == '/' {
		stack = stack[:len(stack)-1]
	}
	// 转换为字符串
	return &Resource{raw: string(stack)}
}

func (r *Resource) String() string {
	return r.raw
}

// Match reports whether raw matches this resource pattern.
func (r *Resource) Match(raw string) bool {
	p := r.raw

	// 根资源特殊处理
	if p == "/" {
		for i := 0; i < len(raw); i++ {
			if raw[i] != '/' {
				return false
			}
		}

		return true
	}

	pi := 1 // pattern 跳过前导 /
	ti := 0

	for {
		// pattern 已结束
		if pi >= len(p) {
			_, ok := nextSegment(raw, &ti)
			return !ok
		}

		// 读取 pattern segment
		ps := pi

		for pi < len(p) && p[pi] != '/' {
			pi++
		}

		pseg := p[ps:pi]

		if pi < len(p) {
			pi++
		}

		// 深度匹配
		if pseg == "**" {
			return true
		}

		// 读取 target segment
		tseg, ok := nextSegment(raw, &ti)

		if !ok {
			return false
		}

		// 单层匹配
		if pseg != "*" && pseg != tseg {
			return false
		}
	}
}

// nextSegment returns the next normalized segment.
//
// It automatically:
//
//   - skips repeated '/'
//   - ignores trailing '/'
//   - does not allocate
func nextSegment(raw string, pos *int) (string, bool) {
	for *pos < len(raw) && raw[*pos] == '/' {
		*pos++
	}

	if *pos >= len(raw) {
		return "", false
	}

	start := *pos

	for *pos < len(raw) && raw[*pos] != '/' {
		*pos++
	}

	return raw[start:*pos], true
}
