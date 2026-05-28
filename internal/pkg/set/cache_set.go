package set

type CacheSetV interface {
	comparable
	Valid() bool
}

type CacheSet[V CacheSetV] struct {
	data      map[V]struct{}
	garbage   map[V]struct{}
	autoclean bool
}

func NewCacheSet[V CacheSetV](autoclean bool) *CacheSet[V] {
	return &CacheSet[V]{
		data:      make(map[V]struct{}),
		garbage:   make(map[V]struct{}),
		autoclean: autoclean,
	}
}

func (s CacheSet[V]) Add(value V) {
	s.data[value] = struct{}{}
}

func (s CacheSet[V]) Has(value V) bool {
	_, ok := s.data[value]
	return ok
}

func (s CacheSet[V]) Delete(value V) {
	delete(s.data, value)
}

func (c CacheSet[V]) GC() Set[V] {
	c.traverse(func(V) {})
	garbage := c.garbage
	c.garbage = make(map[V]struct{})
	return garbage
}

func (c CacheSet[V]) traverse(fn func(V)) {
	for v := range c.data {
		if v.Valid() {
			fn(v)
			continue
		}
		if !c.autoclean {
			c.garbage[v] = struct{}{}
		}
		delete(c.data, v)
	}
}

func (c CacheSet[V]) traverseIf(fn func(V) bool) {
	for v := range c.data {
		if v.Valid() {
			if fn(v) {
				continue
			} else {
				return
			}
		}
		if !c.autoclean {
			c.garbage[v] = struct{}{}
		}
		delete(c.data, v)
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
