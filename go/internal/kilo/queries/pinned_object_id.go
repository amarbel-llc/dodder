package queries

import "code.linenisgreat.com/dodder/go/internal/bravo/ids"

type pinnedObjectId struct {
	ids.Sigil
	ObjectId
}
