package ids

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/charlie/collections_value"
)

func init() {
	collections_value.RegisterGobValue[TagStruct](nil)
}

type (
	ZettelIdSet        = interfaces.Set[ZettelId]
	ZettelIdMutableSet = interfaces.SetMutable[ZettelId]
)

func MakeZettelIdMutableSet(hs ...ZettelId) ZettelIdMutableSet {
	return ZettelIdMutableSet(
		collections_value.MakeMutableValueSet(nil, hs...),
	)
}
