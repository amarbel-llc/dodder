package store_fs

import (
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"code.linenisgreat.com/dodder/go/lib/delta/collections_value"
)

type (
	CheckedOutSet        = interfaces.Set[*sku.CheckedOut]
	CheckedOutMutableSet = interfaces.SetMutable[*sku.CheckedOut]
)

func MakeCheckedOutMutableSet() CheckedOutMutableSet {
	return collections_value.MakeMutableValueSet[*sku.CheckedOut](
		nil,
	)
}
