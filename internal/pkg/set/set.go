package set

import "maps"

// Set is a generic set backed by map[K]struct{}. It provides O(1) membership
// tests and common set operations. The zero value is NOT usable — use make(Set[K]).
type Set[K comparable] map[K]struct{}

// Len returns the number of elements in the set.
func (s Set[K]) Len() int {
	return len(s)
}

// Has reports whether key is in the set.
func (s Set[K]) Has(key K) bool {
	_, ok := s[key]
	return ok
}

// Set inserts key into the set (mutates receiver).
func (s Set[K]) Set(key K) {
	s[key] = struct{}{}
}

// Delete removes key from the set (no-op if absent).
func (s Set[K]) Delete(key K) {
	delete(s, key)
}

// Clear removes all elements.
func (s Set[K]) Clear() {
	clear(s)
}

// Copy returns a shallow copy.
func (s Set[K]) Copy() Set[K] {
	out := make(map[K]struct{}, len(s))
	maps.Copy(out, s)
	return out
}

// Slice returns all keys as a slice in no particular order.
func (s Set[K]) Slice() []K {
	out := make([]K, 0)
	for k := range s {
		out = append(out, k)
	}
	return out
}

// Any returns true if fn reports true for at least one element.
func (s Set[K]) Any(fn func(key K) bool) bool {
	for k := range s {
		if fn(k) {
			return true
		}
	}
	return false
}

// All returns true if fn reports true for every element.
func (s Set[K]) All(fn func(key K) bool) bool {
	for k := range s {
		if !fn(k) {
			return false
		}
	}
	return true
}

// Range calls fn once for each element.
func (s Set[K]) Range(fn func(key K)) {
	for k := range s {
		fn(k)
	}
}

// Filter returns a new set containing elements for which fn returns true.
func (s Set[K]) Filter(fn func(key K) bool) Set[K] {
	out := make(map[K]struct{})
	for k := range s {
		if fn(k) {
			out[k] = struct{}{}
		}
	}
	return out
}

// ExtractIf removes elements matching fn from the receiver and returns them in a new set.
func (s Set[K]) ExtractIf(fn func(key K) bool) Set[K] {
	out := make(map[K]struct{})
	for k := range s {
		if fn(k) {
			delete(s, k)
			out[k] = struct{}{}
		}
	}
	return out
}

// Add_ merges s2 into the receiver (mutates receiver, same as union).
func (s Set[K]) Add_(s2 Set[K]) {
	maps.Copy(s, s2)
}

// Add returns a new set that is the union of s and s2 (does not mutate receiver).
func (s Set[K]) Add(s2 Set[K]) Set[K] {
	out := make(map[K]struct{}, len(s)+len(s2))
	maps.Copy(out, s)
	maps.Copy(out, s2)
	return out
}

// Sub_ removes all elements of s2 from the receiver (mutates receiver).
func (s Set[K]) Sub_(s2 Set[K]) {
	for k := range s2 {
		delete(s, k)
	}
}
