//go:build !debug

package pool

import "code.linenisgreat.com/dodder/go/lib/_/interfaces"

func wrapRepoolDebug(repool interfaces.FuncRepool) interfaces.FuncRepool {
	return repool
}

func OutstandingBorrows() int64 {
	return 0
}
