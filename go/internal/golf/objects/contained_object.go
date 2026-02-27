package objects

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/cmp"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
)

type (
	SeqId = ids.SeqId

	// required to be exported for Gob's stupid illusions
	// TODO rename maybe to lock entry?
	containedObject struct {
		ContainedObjectType ContainedObjectType
		Alias               SeqId
		Lock                markl.Lock[SeqId, *SeqId]
	}
)

func (object containedObject) GetKey() SeqId {
	return object.Lock.GetKey()
}

func containedObjectCompareKey(left, right containedObject) cmp.Result {
	return ids.SeqIdCompare(left.GetKey(), right.GetKey())
}
