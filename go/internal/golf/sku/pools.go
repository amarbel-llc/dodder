package sku

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
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
