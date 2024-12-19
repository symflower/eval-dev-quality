package util

// Set creates a set from a slice.
func Set[T comparable](s []T) map[T]bool {
	set := make(map[T]bool, len(s))
	for _, i := range s {
		set[i] = true
	}

	return set
}
