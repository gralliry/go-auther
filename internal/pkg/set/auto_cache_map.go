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

func (s *AutoCacheMap[K, V]) Add(val V) {
	s.data[val.ID()] = val
}

func (s *AutoCacheMap[K, V]) Get(key K) (V, bool) {
	val, exist := s.data[key]
	if exist {
		if val.Valid() {
			return val, true
		}
		delete(s.data, key)
	}
	return val, false
}

func (s *AutoCacheMap[K, V]) HasByKey(key K) bool {
	if val, exist := s.data[key]; exist {
		if val.Valid() {
			return true
		}
		delete(s.data, key)
	}
	return false
}

func (s *AutoCacheMap[K, V]) HasByVal(val V) bool {
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

func (s *AutoCacheMap[K, V]) DeleteByKey(key K) {
	delete(s.data, key)
}

func (s *AutoCacheMap[K, V]) DeleteByVal(val V) {
	oldValue, exist := s.data[val.ID()]
	if exist && (oldValue == val || !oldValue.Valid()) {
		delete(s.data, val.ID())
	}
}

func (s *AutoCacheMap[K, V]) traverse(fn func(V)) {
	for key, val := range s.data {
		if !val.Valid() {
			delete(s.data, key)
		}
		fn(val)
	}
}

func (s *AutoCacheMap[K, V]) traverseIf(fn func(V) bool) {
	for key, val := range s.data {
		if !val.Valid() {
			delete(s.data, key)
			continue
		}
		if !fn(val) {
			return
		}
	}
}

func (s *AutoCacheMap[K, V]) Length() int {
	count := 0
	s.traverse(func(V) {
		count++
	})
	return count
}

func (s *AutoCacheMap[K, V]) ToSlice() []V {
	out := make([]V, 0)
	s.traverse(func(v V) {
		out = append(out, v)
	})
	return out
}

func (s *AutoCacheMap[K, V]) Filter(fn func(V) bool) AutoCacheMap[K, V] {
	out := make(map[K]V)
	s.traverse(func(val V) {
		if fn(val) {
			out[val.ID()] = val
		}
	})
	return AutoCacheMap[K, V]{
		data: out,
	}
}

func (s *AutoCacheMap[K, V]) Range(fn func(V)) {
	s.traverse(fn)
}

func (s *AutoCacheMap[K, V]) All(fn func(V) bool) bool {
	ok := true
	s.traverseIf(func(val V) bool {
		ok = fn(val)
		return ok
	})
	return ok
}

func (s *AutoCacheMap[K, V]) Any(fn func(V) bool) bool {
	found := false
	s.traverseIf(func(val V) bool {
		found = fn(val)
		return !found
	})
	return found
}
