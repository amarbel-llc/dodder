package sku

import (
	"encoding/gob"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

var (
	transactedKeyerObjectId   ObjectIdKeyer[*Transacted]
	externalLikeKeyerObjectId = GetExternalLikeKeyer[ExternalLike]()
	CheckedOutKeyerObjectId   = GetExternalLikeKeyer[*CheckedOut]()
)

func init() {
	gob.Register(transactedKeyerObjectId)
}

func GetExternalLikeKeyer[
	ELEMENT interface {
		ExternalObjectIdGetter
		ids.ObjectIdGetter
		ExternalLikeGetter
	},
]() interfaces.StringKeyer[ELEMENT] {
	return interfaces.CompoundKeyer[ELEMENT]{
		ObjectIdKeyer[ELEMENT]{},
		ExternalObjectIdKeyer[ELEMENT]{},
		DescriptionKeyer[ELEMENT]{},
	}
}

type ObjectIdKeyer[ELEMENT ids.ObjectIdGetter] struct{}

func (keyer ObjectIdKeyer[ELEMENT]) GetKey(element ELEMENT) (key string) {
	if element.GetObjectId().IsEmpty() {
		return key
	}

	key = element.GetObjectId().String()

	return key
}

type ExternalObjectIdKeyer[ELEMENT ExternalObjectIdGetter] struct{}

func (ExternalObjectIdKeyer[ELEMENT]) GetKey(element ELEMENT) (key string) {
	if element.GetExternalObjectId().IsEmpty() {
		return key
	}

	key = element.GetExternalObjectId().String()

	return key
}

type DescriptionKeyer[ELEMENT ExternalLikeGetter] struct{}

func (DescriptionKeyer[ELEMENT]) GetKey(element ELEMENT) (key string) {
	key = element.GetSkuExternal().Metadata.GetDescription().String()
	return key
}
