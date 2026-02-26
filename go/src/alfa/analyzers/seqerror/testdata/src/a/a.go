package a

import "iter"

func makeSeq() iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {}
}

// --- Rule 1: blank error variable ---

func blankError() {
	for x, _ := range makeSeq() { // want "error from iter.Seq2 range is discarded"
		_ = x
	}
}
