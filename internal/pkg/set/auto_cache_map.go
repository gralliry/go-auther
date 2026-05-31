package set

type AutoCacheMapNode[K comparable] = interface {
	comparable
	ID() K
	Valid() bool
}

type AutoCacheMap[K comparable, V AutoCacheMapNode[K]] struct {
	data map[K]V
}

func NewAutoCacheMap[K comparable, V AutoCacheMapNode[K]]() *AutoCacheMap[K, V] {
	return &AutoCacheMap[K, V]{
		data: make(map[K]V),
	}
}

func (s AutoCacheMap[K, V]) Add(val V) {
	s.data[val.ID()] = val
}

func (s AutoCacheMap[K, V]) Get(key K) (V, bool) {
	val, exist := s.data[key]
	if exist {
		if val.Valid() {
			return val, true
		}
		delete(s.data, key)
	}
	return val, false
}

func (s AutoCacheMap[K, V]) HasByKey(key K) bool {
	if val, exist := s.data[key]; exist {
		if val.Valid() {
			return true
		}
		delete(s.data, key)
	}
	return false
}

func (s AutoCacheMap[K, V]) HasByVal(val V) bool {
	oldValue, exist := s.data[val.ID()]
	if !exist {
		return false
	}
	if !oldValue.Valid() {
		delete(s.data, val.ID())
		return false
	}
	return oldValue == val
}

func (s AutoCacheMap[K, V]) DeleteByKey(key K) {
	delete(s.data, key)
}

func (s AutoCacheMap[K, V]) DeleteByVal(val V) {
	oldValue, exist := s.data[val.ID()]
	if exist && (oldValue == val || !oldValue.Valid()) {
		delete(s.data, val.ID())
	}
}

func (c AutoCacheMap[K, V]) traverse(fn func(V)) {
	for key, val := range c.data {
		if !val.Valid() {
			delete(c.data, key)
		}
		fn(val)
	}
}

func (c AutoCacheMap[K, V]) traverseIf(fn func(V) bool) {
	for key, val := range c.data {
		if !val.Valid() {
			delete(c.data, key)
			continue
		}
		if !fn(val) {
			return
		}
	}
}

func (c AutoCacheMap[K, V]) Length() int {
	count := 0
	c.traverse(func(V) {
		count++
	})
	return count
}

func (c AutoCacheMap[K, V]) ToSlice() []V {
	out := make([]V, 0)
	c.traverse(func(v V) {
		out = append(out, v)
	})
	return out
}

func (c AutoCacheMap[K, V]) Filter(fn func(V) bool) AutoCacheMap[K, V] {
	out := make(map[K]V)
	c.traverse(func(val V) {
		if fn(val) {
			out[val.ID()] = val
		}
	})
	return AutoCacheMap[K, V]{
		data: out,
	}
}

func (c AutoCacheMap[K, V]) Range(fn func(V)) {
	c.traverse(fn)
}

func (c AutoCacheMap[K, V]) All(fn func(V) bool) bool {
	ok := true
	c.traverseIf(func(val V) bool {
		ok = fn(val)
		return ok
	})
	return ok
}

func (c AutoCacheMap[K, V]) Any(fn func(V) bool) bool {
	found := false
	c.traverseIf(func(val V) bool {
		found = fn(val)
		return !found
	})
	return found
}
