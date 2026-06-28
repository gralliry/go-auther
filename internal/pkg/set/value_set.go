package set

import (
	"iter"
	"maps"
)

// ValueSetNode constrains values stored in a ValueSet: they must be comparable
// and expose an ID() key for the map backing.
type ValueSetNode[K comparable] interface {
	comparable
	ID() K
}

// ValueSet is a generic map[K]V where K = V.ID(). It provides O(1) lookup by
// key or value identity. No auto-clean on traversal — callers manage validity.
type ValueSet[K comparable, V ValueSetNode[K]] map[K]V

// Length returns the number of entries.
func (i ValueSet[K, V]) Length() int {
	return len(i)
}

// HasKey reports whether the key exists in the set.
func (i ValueSet[K, V]) HasKey(key K) bool {
	_, ok := i[key]
	return ok
}

// HasValue reports whether the exact value exists (checked by ID and identity).
func (i ValueSet[K, V]) HasValue(value V) bool {
	existValue, exists := i[value.ID()]
	return exists && existValue == value
}

// Get retrieves the value for the given key.
func (i ValueSet[K, V]) Get(key K) (V, bool) {
	value, ok := i[key]
	return value, ok
}

// Add inserts or replaces the value, keyed by value.ID().
func (i ValueSet[K, V]) Add(value V) {
	i[value.ID()] = value
}

// DeleteByKey removes the entry for the given key.
func (i ValueSet[K, V]) DeleteByKey(key K) {
	delete(i, key)
}

// DeleteByValue removes the entry matching value.ID().
func (i ValueSet[K, V]) DeleteByValue(value V) {
	delete(i, value.ID())
}

// Clear removes all entries.
func (i ValueSet[K, V]) Clear() {
	clear(i)
}

// ToSlice returns all values in no particular order.
func (i ValueSet[K, V]) ToSlice() []V {
	out := make([]V, 0, len(i))
	for _, v := range i {
		out = append(out, v)
	}
	return out
}

// ToMap returns a shallow copy of the underlying map.
func (i ValueSet[K, V]) ToMap() map[K]V {
	out := make(map[K]V, len(i))
	maps.Copy(out, i)
	return out
}

// Copy returns a shallow copy of the set.
func (i ValueSet[K, V]) Copy() ValueSet[K, V] {
	out := make(ValueSet[K, V], len(i))
	maps.Copy(out, i)
	return out
}

// Values returns an iterator over the values, usable with range-over-func.
func (i ValueSet[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, v := range i {
			if !yield(v) {
				return
			}
		}
	}
}

// Entries returns an iterator over (key, value) pairs.
func (i ValueSet[K, V]) Entries() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for k, v := range i {
			if !yield(k, v) {
				return
			}
		}
	}
}

// Range calls f once for each value.
func (i ValueSet[K, V]) Range(f func(V)) {
	for _, v := range i {
		f(v)
	}
}

// RemoveIf deletes values matching f and returns the count removed.
func (i ValueSet[K, V]) RemoveIf(f func(V) bool) int {
	reduced := 0
	for k, v := range i {
		if f(v) {
			delete(i, k)
			reduced++
		}
	}
	return reduced

}

// ExtractIf removes matching values from receiver and returns them in a new set.
func (i ValueSet[K, V]) ExtractIf(f func(V) bool) ValueSet[K, V] {
	out := make(map[K]V)
	for k, v := range i {
		if f(v) {
			delete(i, k)
			out[k] = v
		}
	}
	return out
}

// Filter returns a new set of values for which f returns true.
func (i ValueSet[K, V]) Filter(f func(V) bool) ValueSet[K, V] {
	out := make(map[K]V)
	for k, v := range i {
		if f(v) {
			out[k] = v
		}
	}
	return out
}

// All returns true if f returns true for every value.
func (i ValueSet[K, V]) All(f func(V) bool) bool {
	for _, v := range i {
		if !f(v) {
			return false
		}
	}
	return true
}

// Any returns true if f returns true for at least one value.
func (i ValueSet[K, V]) Any(f func(V) bool) bool {
	for _, v := range i {
		if f(v) {
			return true
		}
	}
	return false
}
