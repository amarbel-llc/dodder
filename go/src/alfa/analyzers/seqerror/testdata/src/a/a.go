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

func blankErrorSuppressed() {
	for x, _ := range makeSeq() { //seq:err-checked
		_ = x
	}
}

// --- Rule 2: named but unchecked error ---

func namedButUnchecked() {
	for x, err := range makeSeq() { // want `error variable "err" from iter.Seq2 range is never checked or propagated`
		_ = err
		_ = x
	}
}

func namedButEmptyCheck() {
	for x, err := range makeSeq() { // want `error variable "err" is checked but not handled`
		if err != nil {
		}
		_ = x
	}
}
