package store_fs

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/collections_value"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
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
