package set

type AuthCacheSetNode interface {
	comparable
	Valid() bool
}

type AutoCacheSet[V AuthCacheSetNode] struct {
	data map[V]struct{}
}

func NewAutoCacheSet[V AuthCacheSetNode]() *AutoCacheSet[V] {
	return &AutoCacheSet[V]{
		data: make(map[V]struct{}),
	}
}

func (s *AutoCacheSet[V]) Add(val V) {
	s.data[val] = struct{}{}
}

func (s *AutoCacheSet[V]) Has(val V) bool {
	_, ok := s.data[val]
	return ok
}

func (s *AutoCacheSet[V]) Delete(val V) {
	delete(s.data, val)
}

func (s *AutoCacheSet[V]) Clear() {
	s.data = make(map[V]struct{})
}

func (s *AutoCacheSet[V]) traverse(fn func(V)) {
	for v := range s.data {
		if !v.Valid() {
			delete(s.data, v)
			continue
		}
		fn(v)
	}
}

func (s *AutoCacheSet[V]) traverseIf(fn func(V) bool) {
	for v := range s.data {
		if !v.Valid() {
			delete(s.data, v)
			continue
		}
		if !fn(v) {
			return
		}
	}
}

func (s *AutoCacheSet[V]) Length() int {
	count := 0
	s.traverse(func(V) {
		count++
	})
	return count
}

func (s *AutoCacheSet[V]) ToSlice() []V {
	out := make([]V, 0)
	s.traverse(func(v V) {
		out = append(out, v)
	})
	return out
}

func (s *AutoCacheSet[V]) Filter(fn func(V) bool) *AutoCacheSet[V] {
	out := map[V]struct{}{}
	s.traverse(func(v V) {
		if fn(v) {
			out[v] = struct{}{}
		}
	})
	return &AutoCacheSet[V]{
		data: out,
	}
}

func (s *AutoCacheSet[V]) Range(fn func(V)) {
	s.traverse(fn)
}

func (s *AutoCacheSet[V]) Count(fn func(V) bool) int {
	num := 0
	s.traverse(func(v V) {
		if fn(v) {
			num += 1
		}
	})
	return num
}

func (s *AutoCacheSet[V]) First(fn func(V) bool) (bool, V) {
	var (
		ok    bool = false
		value V
	)
	s.traverseIf(func(v V) bool {
		if fn(v) {
			ok = true
			value = v
			return false
		}
		return true
	})
	return ok, value
}

func (s *AutoCacheSet[V]) All(fn func(V) bool) bool {
	ok := true
	s.traverseIf(func(v V) bool {
		ok = fn(v)
		return ok
	})
	return ok
}

func (s *AutoCacheSet[V]) Any(fn func(V) bool) bool {
	found := false
	s.traverseIf(func(v V) bool {
		found = fn(v)
		return !found
	})
	return found
}
