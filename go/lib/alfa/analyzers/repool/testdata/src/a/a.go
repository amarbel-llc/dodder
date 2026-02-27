package a

import "a/interfaces"

type fakePool struct{}

func (fakePool) GetWithRepool() (string, interfaces.FuncRepool) {
	return "", func() {}
}

var pool fakePool

func discardedBlank() {
	_, _ = pool.GetWithRepool() // want "the repool function returned by GetWithRepool should be called, not discarded, to avoid a pool leak"
}

func discardedBlankWithOwned() {
	_, _ = pool.GetWithRepool() //repool:owned
}

func deferRepool() {
	_, repool := pool.GetWithRepool()
	defer repool()
}

func directCall() {
	_, repool := pool.GetWithRepool()
	repool()
}

func conditionalCall() {
	v, repool := pool.GetWithRepool() // want "the repool function is not called on all paths"
	if v == "x" {
		repool()
	}
}

func passedToFunction() {
	_, repool := pool.GetWithRepool()
	consume(repool)
}

func consume(fn interfaces.FuncRepool) {
	fn()
}

func assignedToStruct() {
	type holder struct {
		fn interfaces.FuncRepool
	}

	_, repool := pool.GetWithRepool()
	h := holder{fn: repool}
	_ = h
}

func multipleReturns() {
	_, repool := pool.GetWithRepool()
	if true {
		repool()
		return
	}
	repool()
}
