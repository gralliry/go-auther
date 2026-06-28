package set

// CacheSetNode constrains values in a CacheSet: comparable with a Valid() method.
type CacheSetNode interface {
	comparable
	Valid() bool
}

// CacheSet is a set that defers garbage collection. Invalid entries are moved
// to a garbage set on traversal and returned via GC() for explicit cleanup.
// Unlike AutoCacheSet, callers control when garbage is collected.
type CacheSet[V CacheSetNode] struct {
	data    map[V]struct{} // active entries
	garbage map[V]struct{} // invalid entries collected during traversal
}

// NewCacheSet creates an initialized CacheSet.
func NewCacheSet[V CacheSetNode]() *CacheSet[V] {
	return &CacheSet[V]{
		data:    make(map[V]struct{}),
		garbage: make(map[V]struct{}),
	}
}

// Add inserts a value into the set.
func (s *CacheSet[V]) Add(val V) {
	s.data[val] = struct{}{}
}

// Has reports whether the value is present (does NOT check validity).
func (s *CacheSet[V]) Has(val V) bool {
	_, ok := s.data[val]
	return ok
}

// Delete removes a value directly (no GC involved).
func (s *CacheSet[V]) Delete(val V) {
	delete(s.data, val)
}

// GC performs a full traversal to find invalid entries, returns them, and resets
// the garbage set. Call this when you want to release invalid references.
func (s *CacheSet[V]) GC() Set[V] {
	s.traverse(func(V) {})
	garbage := s.garbage
	s.garbage = make(map[V]struct{})
	return garbage
}

// traverse iterates all entries, moving invalid ones to garbage and skipping them.
// Valid entries are passed to fn.
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

// traverseIf is like traverse but stops early if fn returns false.
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

// Length returns the count of valid entries.
func (s *CacheSet[V]) Length() int {
	count := 0
	s.traverse(func(V) {
		count++
	})
	return count
}

// ToSlice returns all valid entries as a slice.
func (s *CacheSet[V]) ToSlice() []V {
	out := make([]V, 0)
	s.traverse(func(v V) {
		out = append(out, v)
	})
	return out
}

// Filter returns a new CacheSet with only valid entries matching fn.
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

// Range calls fn for each valid entry.
func (s *CacheSet[V]) Range(fn func(V)) {
	s.traverse(fn)
}

// All returns true if fn returns true for every valid entry.
func (s *CacheSet[V]) All(fn func(V) bool) bool {
	ok := true
	s.traverseIf(func(v V) bool {
		ok = fn(v)
		return ok
	})
	return ok
}

// Any returns true if fn returns true for at least one valid entry.
func (s *CacheSet[V]) Any(fn func(V) bool) bool {
	found := false
	s.traverseIf(func(v V) bool {
		found = fn(v)
		return !found
	})
	return found
}
