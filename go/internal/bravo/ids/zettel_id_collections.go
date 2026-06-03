package ids

import (
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/collections_value"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
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
