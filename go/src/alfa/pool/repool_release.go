//go:build !debug

package pool

import "code.linenisgreat.com/dodder/go/src/_/interfaces"

func wrapRepoolDebug(repool interfaces.FuncRepool) interfaces.FuncRepool {
	return repool
}

func OutstandingBorrows() int64 {
	return 0
}
