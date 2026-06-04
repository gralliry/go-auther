package set

type CacheSetNode interface {
	comparable
	Valid() bool
}

type CacheSet[V CacheSetNode] struct {
	data    map[V]struct{}
	garbage map[V]struct{}
}

func NewCacheSet[V CacheSetNode]() *CacheSet[V] {
	return &CacheSet[V]{
		data:    make(map[V]struct{}),
		garbage: make(map[V]struct{}),
	}
}

func (s *CacheSet[V]) Add(val V) {
	s.data[val] = struct{}{}
}

func (s *CacheSet[V]) Has(val V) bool {
	_, ok := s.data[val]
	return ok
}

func (s *CacheSet[V]) Delete(val V) {
	delete(s.data, val)
}

func (s *CacheSet[V]) GC() Set[V] {
	s.traverse(func(V) {})
	garbage := s.garbage
	s.garbage = make(map[V]struct{})
	return garbage
}

func (s *CacheSet[V]) traverse(fn func(V)) {
	for v := range s.data {
		if !v.Valid() {
			s.garbage[v] = struct{}{}
			delete(s.data, v)
			continue
		}
		fn(v)
	}
}

func (s *CacheSet[V]) traverseIf(fn func(V) bool) {
	for v := range s.data {
		if !v.Valid() {
			s.garbage[v] = struct{}{}
			delete(s.data, v)
			continue
		}
		if !fn(v) {
			return
		}
	}
}

func (s *CacheSet[V]) Length() int {
	count := 0
	s.traverse(func(V) {
		count++
	})
	return count
}

func (s *CacheSet[V]) ToSlice() []V {
	out := make([]V, 0)
	s.traverse(func(v V) {
		out = append(out, v)
	})
	return out
}

func (s *CacheSet[V]) Filter(fn func(V) bool) CacheSet[V] {
	out := map[V]struct{}{}
	s.traverse(func(v V) {
		if fn(v) {
			out[v] = struct{}{}
		}
	})
	return CacheSet[V]{
		data: out,
	}
}

func (s *CacheSet[V]) Range(fn func(V)) {
	s.traverse(fn)
}

func (s *CacheSet[V]) All(fn func(V) bool) bool {
	ok := true
	s.traverseIf(func(v V) bool {
		ok = fn(v)
		return ok
	})
	return ok
}

func (s *CacheSet[V]) Any(fn func(V) bool) bool {
	found := false
	s.traverseIf(func(v V) bool {
		found = fn(v)
		return !found
	})
	return found
}
