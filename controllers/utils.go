package controllers

import (
	"k8s.io/utils/strings/slices"
)

// isEqualUnordered compares two string slices and returns true if they contain the same
// elements, regardless of order. Returns false otherwise, or if they are of different length.
func isEqualUnordered(s1 []string, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}

	for _, elem := range s1 {
		if !slices.Contains(s2, elem) {
			return false
		}
	}

	return true
}
