package store_fs

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/charlie/collections_value"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
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
