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

// --- Valid patterns (no diagnostics expected) ---

func checkedWithReturn() (string, error) {
	for x, err := range makeSeq() {
		if err != nil {
			return "", err
		}
		_ = x
	}
	return "", nil
}

func checkedWithContinue() {
	for x, err := range makeSeq() {
		if err != nil {
			continue
		}
		_ = x
	}
}

func checkedWithBreak() {
	for x, err := range makeSeq() {
		if err != nil {
			break
		}
		_ = x
	}
}

func checkedWithPanic() {
	for x, err := range makeSeq() {
		if err != nil {
			panic(err)
		}
		_ = x
	}
}

func checkedWithFuncCall() {
	for x, err := range makeSeq() {
		if err != nil {
			handleErr(err)
			return
		}
		_ = x
	}
}

func handleErr(error) {}

func yieldPassThrough() {
	_ = func(yield func(string, error) bool) {
		for x, err := range makeSeq() {
			if !yield(x, err) {
				return
			}
		}
	}
}

func yieldWithNilCheck() {
	_ = func(yield func(string, error) bool) {
		for x, err := range makeSeq() {
			if err != nil {
				yield("", err)
				return
			}
			if !yield(x, nil) {
				return
			}
		}
	}
}

// --- Non-error sequences (no diagnostics expected) ---

func nonErrorSeq2() {
	seq := func(yield func(string, int) bool) {}
	for x, i := range seq {
		_ = x
		_ = i
	}
}

func plainSeq() {
	seq := func(yield func(string) bool) {}
	for x := range seq {
		_ = x
	}
}
