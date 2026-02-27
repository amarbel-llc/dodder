package queries

import "code.linenisgreat.com/dodder/go/internal/foxtrot/ids"

type pinnedObjectId struct {
	ids.Sigil
	ObjectId
}
