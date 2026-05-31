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

func (s AutoCacheSet[V]) Add(val V) {
	s.data[val] = struct{}{}
}

func (s AutoCacheSet[V]) Has(val V) bool {
	_, ok := s.data[val]
	return ok
}

func (s AutoCacheSet[V]) Delete(val V) {
	delete(s.data, val)
}

func (s AutoCacheSet[V]) Clear() {
	s.data = make(map[V]struct{})
}

func (c AutoCacheSet[V]) traverse(fn func(V)) {
	for v := range c.data {
		if !v.Valid() {
			delete(c.data, v)
			continue
		}
		fn(v)
	}
}

func (c AutoCacheSet[V]) traverseIf(fn func(V) bool) {
	for v := range c.data {
		if !v.Valid() {
			delete(c.data, v)
			continue
		}
		if !fn(v) {
			return
		}
	}
}

func (c AutoCacheSet[V]) Length() int {
	count := 0
	c.traverse(func(V) {
		count++
	})
	return count
}

func (c AutoCacheSet[V]) ToSlice() []V {
	out := make([]V, 0)
	c.traverse(func(v V) {
		out = append(out, v)
	})
	return out
}

func (c AutoCacheSet[V]) Filter(fn func(V) bool) AutoCacheSet[V] {
	out := map[V]struct{}{}
	c.traverse(func(v V) {
		if fn(v) {
			out[v] = struct{}{}
		}
	})
	return AutoCacheSet[V]{
		data: out,
	}
}

func (c AutoCacheSet[V]) Range(fn func(V)) {
	c.traverse(fn)
}

func (c AutoCacheSet[V]) All(fn func(V) bool) bool {
	ok := true
	c.traverseIf(func(v V) bool {
		ok = fn(v)
		return ok
	})
	return ok
}

func (c AutoCacheSet[V]) Any(fn func(V) bool) bool {
	found := false
	c.traverseIf(func(v V) bool {
		found = fn(v)
		return !found
	})
	return found
}
