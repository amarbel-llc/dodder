package ids

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/delta/collections_value"
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
