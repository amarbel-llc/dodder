package store_fs

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

var (
	poolExternal   interfaces.PoolPtr[sku.Transacted, *sku.Transacted]
	poolCheckedOut interfaces.PoolPtr[sku.CheckedOut, *sku.CheckedOut]
)

func init() {
	poolExternal = pool.Make[sku.Transacted](
		nil,
		nil,
	)

	poolCheckedOut = pool.Make[sku.CheckedOut](
		nil,
		nil,
	)
}

func GetExternalPool() interfaces.PoolPtr[sku.Transacted, *sku.Transacted] {
	return poolExternal
}

func GetCheckedOutPool() interfaces.PoolPtr[sku.CheckedOut, *sku.CheckedOut] {
	return poolCheckedOut
}
