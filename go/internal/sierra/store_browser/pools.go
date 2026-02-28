//go:build chrest

package store_browser

import (
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
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
