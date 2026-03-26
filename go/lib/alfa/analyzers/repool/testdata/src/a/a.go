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

func suppressedConditional() {
	v, repool := pool.GetWithRepool() //repool:suppress ownership transfer
	if v == "x" {
		repool()
	}
}

func suppressedWithIssueLink() {
	v, repool := pool.GetWithRepool() //repool:suppress #47 false positive: nil-guarded defer
	if v == "x" {
		repool()
	}
}

func unsuppressedConditional() {
	v, repool := pool.GetWithRepool() // want "the repool function is not called on all paths"
	if v == "y" {
		repool()
	}
}

// Pattern 1: panic terminates — no leak possible
func panicOnErrorPath() {
	_, repool := pool.GetWithRepool()
	if true {
		repool()
		return
	}
	panic("unreachable")
}

// Pattern 2: repool captured in returned closure
func returnedInClosure() (string, func()) {
	v, repool := pool.GetWithRepool()
	return v, func() {
		repool()
	}
}

// Pattern 1b: panic on SOME paths but return without repool on others — still a leak
func panicOnSomePathsButNotAll() {
	v, repool := pool.GetWithRepool() // want "the repool function is not called on all paths"
	if v == "x" {
		repool()
		return
	}
	if v == "y" {
		panic("error")
	}
	// implicit return without repool — leak
}

// Pattern 3: nil-guarded defer before assignment
func nilGuardedDefer() {
	var repool interfaces.FuncRepool

	defer func() {
		if repool != nil {
			repool()
		}
	}()

	_, repool = pool.GetWithRepool()
}
