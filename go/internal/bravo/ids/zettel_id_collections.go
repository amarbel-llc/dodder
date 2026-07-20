package ids

import (
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/collections_value"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

type (
	ZettelIdSet        = interfaces.Set[ZettelId]
	ZettelIdMutableSet = interfaces.SetMutable[ZettelId]
)

func MakeZettelIdMutableSet(hs ...ZettelId) ZettelIdMutableSet {
	return ZettelIdMutableSet(
		collections_value.MakeMutableValueSet(nil, hs...),
	)
}
