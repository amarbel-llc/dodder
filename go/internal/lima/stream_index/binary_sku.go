package stream_index

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ohio"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
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
