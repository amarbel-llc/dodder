package queries

import (
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
)

// TODO abstract into "hoisted" ids
type expObjectIds struct {
	internal map[string]ObjectId
	external map[string]sku.ExternalObjectId
}

func (oids expObjectIds) Len() int {
	return len(oids.internal) + len(oids.external)
}

func (oids expObjectIds) IsEmpty() bool {
	if len(oids.internal) > 0 {
		return false
	}

	if len(oids.external) > 0 {
		return false
	}

	return true
}
