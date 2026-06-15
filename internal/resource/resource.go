package resource

import "strings"

type Resource struct {
	segs []string
}

func NewResource(raw string) *Resource {
	var segs []string
	n := len(raw)
	s := 0
	for i := range n {
		switch raw[i] {
		case '/':
			if s < i {
				segs = append(segs, raw[s:i])
			}
			s = i + 1
		case '*':
			// **
			if i+1 < n && raw[i+1] == '*' {
				if s < i {
					segs = append(segs, raw[s:i])
				}
				segs = append(segs, raw[i:i+2])
				return &Resource{segs: segs}
			}
			// single *
			if s < i {
				segs = append(segs, raw[s:i])
			}
			segs = append(segs, raw[i:i+1])
			s = i + 1
		}
	}
	if s < n {
		segs = append(segs, raw[s:n])
	}
	return &Resource{segs: segs}
}

func (r *Resource) String() string {
	return "/" + strings.Join(r.segs, "/")
}

func (r *Resource) Contains(r2 *Resource) bool {
	i, j := 0, 0
	n, m := len(r.segs), len(r2.segs)
	for i < n && j < m {
		p := r.segs[i]
		t := r2.segs[j]
		switch p {
		case "**":
			return true
		case "*":
			i++
			j++
			continue
		default:
			if p != t {
				return false
			}
			i++
			j++
		}
	}
	// pattern ended but target has more segments
	if i == n && j < m {
		return false
	}
	// target ended but pattern has remaining segments
	for i < n {
		if r.segs[i] != "**" {
			return false
		}
		i++
	}
	return true
}

func (r *Resource) Match(raw string) bool {
	i, j := 0, 0
	n := len(raw)
	pn := len(r.segs)

	var buf []byte

	for i < n {
		c := raw[i]

		switch c {
		case '/':
			if len(buf) > 0 {
				if j >= pn {
					return false
				}
				seg := r.segs[j]
				if !matchSeg(seg, string(buf)) {
					return false
				}
				if seg == "**" {
					return true
				}
				j++
				buf = buf[:0]
			}
			i++
		default:
			buf = append(buf, c)
			i++
		}
	}

	// flush last segment
	if len(buf) > 0 {
		if j >= pn {
			return false
		}
		seg := r.segs[j]
		if !matchSeg(seg, string(buf)) {
			return false
		}
		if seg == "**" {
			return true
		}
		j++
	}

	// pattern remaining
	for j < pn {
		if r.segs[j] != "**" {
			return false
		}
		return true
	}

	return j == pn
}

func matchSeg(pat, seg string) bool {
	switch pat {
	case "**":
		return true
	case "*":
		return true
	default:
		return pat == seg
	}
}
