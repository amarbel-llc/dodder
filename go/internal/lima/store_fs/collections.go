package store_fs

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/collections_value"
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
