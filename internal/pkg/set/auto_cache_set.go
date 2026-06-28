package set

// AuthCacheSetNode constrains values in an AutoCacheSet: comparable with Valid().
type AuthCacheSetNode interface {
	comparable
	Valid() bool
}

// AutoCacheSet is a set that automatically drops invalid entries on every traversal.
// Callers never see stale data — Range, Any, All, etc. all silently skip entries
// where Valid() returns false and delete them.
type AutoCacheSet[V AuthCacheSetNode] struct {
	data map[V]struct{}
}

// NewAutoCacheSet creates an initialized AutoCacheSet.
func NewAutoCacheSet[V AuthCacheSetNode]() *AutoCacheSet[V] {
	return &AutoCacheSet[V]{
		data: make(map[V]struct{}),
	}
}

// Add inserts a value into the set.
func (s *AutoCacheSet[V]) Add(val V) {
	s.data[val] = struct{}{}
}

// Has reports whether the value is present (does NOT check validity).
func (s *AutoCacheSet[V]) Has(val V) bool {
	_, ok := s.data[val]
	return ok
}

// Delete removes a value directly.
func (s *AutoCacheSet[V]) Delete(val V) {
	delete(s.data, val)
}

// Clear drops all entries (replaces backing map).
func (s *AutoCacheSet[V]) Clear() {
	s.data = make(map[V]struct{})
}

// traverse iterates all entries, silently deleting invalid ones and skipping them.
func (s *AutoCacheSet[V]) traverse(fn func(V)) {
	for v := range s.data {
		if !v.Valid() {
			delete(s.data, v)
			continue
		}
		fn(v)
	}
}

// traverseIf is like traverse but stops early if fn returns false.
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

// Length returns the count of valid entries.
func (s *AutoCacheSet[V]) Length() int {
	count := 0
	s.traverse(func(V) {
		count++
	})
	return count
}

// ToSlice returns all valid entries as a slice.
func (s *AutoCacheSet[V]) ToSlice() []V {
	out := make([]V, 0)
	s.traverse(func(v V) {
		out = append(out, v)
	})
	return out
}

// Filter returns a new AutoCacheSet with only valid entries matching fn.
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

// Range calls fn for each valid entry.
func (s *AutoCacheSet[V]) Range(fn func(V)) {
	s.traverse(fn)
}

// Count returns the number of valid entries for which fn returns true.
func (s *AutoCacheSet[V]) Count(fn func(V) bool) int {
	num := 0
	s.traverse(func(v V) {
		if fn(v) {
			num += 1
		}
	})
	return num
}

// First returns the first valid entry for which fn returns true.
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

// All returns true if fn returns true for every valid entry.
func (s *AutoCacheSet[V]) All(fn func(V) bool) bool {
	ok := true
	s.traverseIf(func(v V) bool {
		ok = fn(v)
		return ok
	})
	return ok
}

// Any returns true if fn returns true for at least one valid entry.
func (s *AutoCacheSet[V]) Any(fn func(V) bool) bool {
	found := false
	s.traverseIf(func(v V) bool {
		found = fn(v)
		return !found
	})
	return found
}
