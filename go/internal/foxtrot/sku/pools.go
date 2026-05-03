package sku

import (
	"code.linenisgreat.com/dodder/go/lib/0/pool"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

var (
	poolTransacted = pool.Make(
		nil,
		TransactedResetter.Reset,
	)

	poolCheckedOut = pool.Make(
		nil,
		CheckedOutResetter.Reset,
	)
)

func GetTransactedPool() interfaces.PoolPtr[Transacted, *Transacted] {
	return poolTransacted
}

func GetCheckedOutPool() interfaces.PoolPtr[CheckedOut, *CheckedOut] {
	return poolCheckedOut
}
