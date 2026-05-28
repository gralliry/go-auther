package iset

import (
	"iter"
	"maps"
)

type ValueSetV[K comparable] interface {
	comparable
	ID() K
}

type ValueSet[K comparable, V ValueSetV[K]] map[K]V

func (i ValueSet[K, V]) Length() int {
	return len(i)
}

func (i ValueSet[K, V]) HasKey(key K) bool {
	_, ok := i[key]
	return ok
}

func (i ValueSet[K, V]) HasValue(value V) bool {
	existValue, exists := i[value.ID()]
	return exists && existValue == value
}

func (i ValueSet[K, V]) Get(key K) (V, bool) {
	value, ok := i[key]
	return value, ok
}

func (i ValueSet[K, V]) Add(value V) {
	var zero V
	if value == zero {
		return
	}
	i[value.ID()] = value
}

func (i ValueSet[K, V]) DeleteByKey(key K) {
	delete(i, key)
}

func (i ValueSet[K, V]) DeleteByValue(value V) {
	delete(i, value.ID())
}

func (i ValueSet[K, V]) Clear() {
	clear(i)
}

func (i ValueSet[K, V]) ToSlice() []V {
	out := make([]V, 0, len(i))
	for _, v := range i {
		out = append(out, v)
	}
	return out
}

func (i ValueSet[K, V]) ToMap() map[K]V {
	out := make(map[K]V, len(i))
	maps.Copy(out, i)
	return out
}

func (i ValueSet[K, V]) Copy() ValueSet[K, V] {
	out := make(ValueSet[K, V], len(i))
	maps.Copy(out, i)
	return out
}

func (i ValueSet[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		for _, v := range i {
			if !yield(v) {
				return
			}
		}
	}
}

func (i ValueSet[K, V]) Entries() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for k, v := range i {
			if !yield(k, v) {
				return
			}
		}
	}
}

func (i ValueSet[K, V]) Range(f func(V)) {
	for _, v := range i {
		f(v)
	}
}

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

func (i ValueSet[K, V]) Filter(f func(V) bool) ValueSet[K, V] {
	out := make(map[K]V)
	for k, v := range i {
		if f(v) {
			out[k] = v
		}
	}
	return out
}

func (i ValueSet[K, V]) All(f func(V) bool) bool {
	for _, v := range i {
		if !f(v) {
			return false
		}
	}
	return true
}

func (i ValueSet[K, V]) Any(f func(V) bool) bool {
	for _, v := range i {
		if f(v) {
			return true
		}
	}
	return false
}
