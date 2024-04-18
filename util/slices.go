package util

import (
	"sort"

	"golang.org/x/exp/constraints"
)

// InsertToSortedSlice inserts the given item into an already descending sorted slice.
func InsertToSortedSlice[T constraints.Ordered](slice []T, t T) []T {
	if len(slice) == 0 {
		return []T{t}
	}

	i := sort.Search(len(slice), func(i int) bool {
		return slice[i] >= t
	})

	if i == len(slice) {
		return append(slice, t)
	}

	slice = append(slice[:i+1], slice[i:]...)
	slice[i] = t

	return slice
}
