package stream_index

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
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
