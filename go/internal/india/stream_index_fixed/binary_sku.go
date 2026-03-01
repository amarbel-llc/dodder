package stream_index_fixed

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
)

type objectWithSigil struct {
	*sku.Transacted
	ids.Sigil
}

type objectWithCursorAndSigil struct {
	objectWithSigil
	ohio.Cursor
}

type objectMetaWithCursorAndSigil struct {
	ids.Sigil
	ohio.Cursor
	Tai ids.Tai
}
