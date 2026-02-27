package queries

import "code.linenisgreat.com/dodder/go/internal/echo/ids"

type pinnedObjectId struct {
	ids.Sigil
	ObjectId
}
