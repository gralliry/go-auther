package set

type Set[K comparable] map[K]struct{}

func (s Set[K]) Length() int {
	return len(s)
}

func (s Set[K]) Has(key K) bool {
	_, ok := s[key]
	return ok
}

func (s Set[K]) Add(key K) {
	s[key] = struct{}{}
}

func (s Set[K]) Delete(key K) {
	delete(s, key)
}

func (s Set[K]) Clear() {
	clear(s)
}

func (s Set[K]) ToSlice() []K {
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
			out[k] = struct{}{}
		}
	}
	return out
}
