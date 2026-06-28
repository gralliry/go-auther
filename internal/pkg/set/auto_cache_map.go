package set

// AutoCacheMapNode constrains values in an AutoCacheMap: comparable, keyed by ID(),
// with a Valid() method for automatic garbage collection.
type AutoCacheMapNode[K comparable] = interface {
	comparable
	ID() K
	Valid() bool
}

// AutoCacheMap is a map[K]V where each value exposes an ID() key. Invalid entries
// are silently dropped on all traversals (Range, Any, All, etc.) — callers never
// see stale data.
type AutoCacheMap[K comparable, V AutoCacheMapNode[K]] struct {
	data map[K]V
}

// NewAutoCacheMap creates an initialized AutoCacheMap.
func NewAutoCacheMap[K comparable, V AutoCacheMapNode[K]]() *AutoCacheMap[K, V] {
	return &AutoCacheMap[K, V]{
		data: make(map[K]V),
	}
}

// Add inserts or replaces the value, keyed by val.ID().
func (s *AutoCacheMap[K, V]) Add(val V) {
	s.data[val.ID()] = val
}

// Get retrieves the value for key. If the entry is invalid, it is deleted and
// (zero, false) is returned.
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

// HasByKey reports whether a valid entry exists for the given key.
func (s *AutoCacheMap[K, V]) HasByKey(key K) bool {
	if val, exist := s.data[key]; exist {
		if val.Valid() {
			return true
		}
		delete(s.data, key)
	}
	return false
}

// HasByVal reports whether the exact value exists (checked by ID, validity, and identity).
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

// DeleteByKey removes the entry for the given key.
func (s *AutoCacheMap[K, V]) DeleteByKey(key K) {
	delete(s.data, key)
}

// DeleteByVal removes the entry matching val.ID() only if it is identical to val or invalid.
func (s *AutoCacheMap[K, V]) DeleteByVal(val V) {
	oldValue, exist := s.data[val.ID()]
	if exist && (oldValue == val || !oldValue.Valid()) {
		delete(s.data, val.ID())
	}
}

// traverse iterates all entries, silently deleting invalid ones and skipping them.
func (s *AutoCacheMap[K, V]) traverse(fn func(V)) {
	for key, val := range s.data {
		if !val.Valid() {
			delete(s.data, key)
			continue
		}
		fn(val)
	}
}

// traverseIf is like traverse but stops early if fn returns false.
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

// Length returns the count of valid entries.
func (s *AutoCacheMap[K, V]) Length() int {
	count := 0
	s.traverse(func(V) {
		count++
	})
	return count
}

// ToSlice returns all valid entries as a slice.
func (s *AutoCacheMap[K, V]) ToSlice() []V {
	out := make([]V, 0)
	s.traverse(func(v V) {
		out = append(out, v)
	})
	return out
}

// Filter returns a new AutoCacheMap with only valid entries matching fn.
func (s *AutoCacheMap[K, V]) Filter(fn func(V) bool) *AutoCacheMap[K, V] {
	out := make(map[K]V)
	s.traverse(func(val V) {
		if fn(val) {
			out[val.ID()] = val
		}
	})
	return &AutoCacheMap[K, V]{
		data: out,
	}
}

// Range calls fn for each valid entry.
func (s *AutoCacheMap[K, V]) Range(fn func(V)) {
	s.traverse(fn)
}

// All returns true if fn returns true for every valid entry.
func (s *AutoCacheMap[K, V]) All(fn func(V) bool) bool {
	ok := true
	s.traverseIf(func(val V) bool {
		ok = fn(val)
		return ok
	})
	return ok
}

// Any returns true if fn returns true for at least one valid entry.
func (s *AutoCacheMap[K, V]) Any(fn func(V) bool) bool {
	found := false
	s.traverseIf(func(val V) bool {
		found = fn(val)
		return !found
	})
	return found
}
