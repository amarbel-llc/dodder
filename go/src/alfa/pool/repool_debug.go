//go:build debug

package pool

import (
	"fmt"
	"runtime"
	"sync/atomic"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
)

var outstandingBorrows atomic.Int64

func wrapRepoolDebug(repool interfaces.FuncRepool) interfaces.FuncRepool {
	outstandingBorrows.Add(1)

	var called atomic.Bool
	_, file, line, _ := runtime.Caller(1)
	caller := fmt.Sprintf("%s:%d", file, line)

	return func() {
		if !called.CompareAndSwap(false, true) {
			panic(fmt.Sprintf("repool: double-repool detected (originally borrowed at %s)", caller))
		}

		outstandingBorrows.Add(-1)
		repool()
	}
}

func OutstandingBorrows() int64 {
	return outstandingBorrows.Load()
}
