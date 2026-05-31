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

func (s CacheSet[V]) Add(val V) {
	s.data[val] = struct{}{}
}

func (s CacheSet[V]) Has(val V) bool {
	_, ok := s.data[val]
	return ok
}

func (s CacheSet[V]) Delete(val V) {
	delete(s.data, val)
}

func (c CacheSet[V]) GC() Set[V] {
	c.traverse(func(V) {})
	garbage := c.garbage
	c.garbage = make(map[V]struct{})
	return garbage
}

func (c CacheSet[V]) traverse(fn func(V)) {
	for v := range c.data {
		if !v.Valid() {
			c.garbage[v] = struct{}{}
			delete(c.data, v)
			continue
		}
		fn(v)
	}
}

func (c CacheSet[V]) traverseIf(fn func(V) bool) {
	for v := range c.data {
		if !v.Valid() {
			c.garbage[v] = struct{}{}
			delete(c.data, v)
			continue
		}
		if !fn(v) {
			return
		}
	}
}

func (c CacheSet[V]) Length() int {
	count := 0
	c.traverse(func(V) {
		count++
	})
	return count
}

func (c CacheSet[V]) ToSlice() []V {
	out := make([]V, 0)
	c.traverse(func(v V) {
		out = append(out, v)
	})
	return out
}

func (c CacheSet[V]) Filter(fn func(V) bool) CacheSet[V] {
	out := map[V]struct{}{}
	c.traverse(func(v V) {
		if fn(v) {
			out[v] = struct{}{}
		}
	})
	return CacheSet[V]{
		data: out,
	}
}

func (c CacheSet[V]) Range(fn func(V)) {
	c.traverse(fn)
}

func (c CacheSet[V]) All(fn func(V) bool) bool {
	ok := true
	c.traverseIf(func(v V) bool {
		ok = fn(v)
		return ok
	})
	return ok
}

func (c CacheSet[V]) Any(fn func(V) bool) bool {
	found := false
	c.traverseIf(func(v V) bool {
		found = fn(v)
		return !found
	})
	return found
}
