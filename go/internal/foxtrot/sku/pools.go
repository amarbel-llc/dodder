package sku

import (
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/pool"
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
