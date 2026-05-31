package set

import "maps"

type Set[K comparable] map[K]struct{}

func (s Set[K]) Len() int {
	return len(s)
}

func (s Set[K]) Has(key K) bool {
	_, ok := s[key]
	return ok
}

func (s Set[K]) Set(key K) {
	s[key] = struct{}{}
}

func (s Set[K]) Delete(key K) {
	delete(s, key)
}

func (s Set[K]) Clear() {
	clear(s)
}

func (s Set[K]) Copy() Set[K] {
	out := make(map[K]struct{}, len(s))
	maps.Copy(out, s)
	return out
}

func (s Set[K]) Slice() []K {
	out := make([]K, 0)
	for k := range s {
		out = append(out, k)
	}
	return out
}

func (s Set[K]) Any(fn func(key K) bool) bool {
	for k := range s {
		if fn(k) {
			return true
		}
	}
	return false
}

func (s Set[K]) All(fn func(key K) bool) bool {
	for k := range s {
		if !fn(k) {
			return false
		}
	}
	return true
}

func (s Set[K]) Range(fn func(key K)) {
	for k := range s {
		fn(k)
	}
}

func (s Set[K]) Filter(fn func(key K) bool) Set[K] {
	out := make(map[K]struct{})
	for k := range s {
		if fn(k) {
			out[k] = struct{}{}
		}
	}
	return out
}

func (s Set[K]) ExtractIf(fn func(key K) bool) Set[K] {
	out := make(map[K]struct{})
	for k := range s {
		if fn(k) {
			delete(s, k)
			out[k] = struct{}{}
		}
	}
	return out
}

func (s Set[K]) Add_(s2 Set[K]) {
	maps.Copy(s, s2)
}

func (s Set[K]) Add(s2 Set[K]) Set[K] {
	out := make(map[K]struct{}, len(s)+len(s2))
	maps.Copy(out, s)
	maps.Copy(out, s2)
	return out
}

func (s Set[K]) Sub_(s2 Set[K]) {
	for k := range s2 {
		delete(s, k)
	}
}
