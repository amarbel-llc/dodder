package store_fs

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/pool"
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
