package objects

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"github.com/amarbel-llc/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/cmp"
)

type (
	SeqId = ids.SeqId

	// required to be exported for Gob's stupid illusions
	// TODO rename maybe to lock entry?
	containedObject struct {
		ContainedObjectType containedObjectType
		Alias               string
		Lock                markl.Lock[SeqId, *SeqId]
	}
)

func (object containedObject) GetKey() SeqId {
	return object.Lock.GetKey()
}

func containedObjectCompareKey(left, right containedObject) cmp.Result {
	return ids.SeqIdCompare(left.GetKey(), right.GetKey())
}
