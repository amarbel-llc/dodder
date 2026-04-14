package sku

import (
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/delta/collections_value"
)

var (
	TransactedSetEmpty TransactedSet
	TransactedLessor   transactedLessorStable
	TransactedEqualer  transactedEqualer
)

type Collection interfaces.Collection[*Transacted]

func init() {
	TransactedSetEmpty = MakeTransactedSet()
}

type (
	TransactedSet        = interfaces.Set[*Transacted]
	TransactedMutableSet = interfaces.SetMutable[*Transacted]

	ExternalLikeSet        = interfaces.Set[ExternalLike]
	ExternalLikeMutableSet = interfaces.SetMutable[ExternalLike]

	CheckedOutSet        = interfaces.Set[*CheckedOut]
	CheckedOutMutableSet = interfaces.SetMutable[*CheckedOut]
)

func MakeTransactedSet() TransactedSet {
	return collections_value.MakeValueSetFromSlice(transactedKeyerObjectId)
}

func MakeTransactedMutableSet() TransactedMutableSet {
	return collections_value.MakeMutableValueSet(transactedKeyerObjectId)
}

func MakeExternalLikeSet() ExternalLikeSet {
	return collections_value.MakeValueSetFromSlice(externalLikeKeyerObjectId)
}

func MakeExternalLikeMutableSet() ExternalLikeMutableSet {
	return collections_value.MakeMutableValueSet(externalLikeKeyerObjectId)
}

func MakeCheckedOutSet() CheckedOutSet {
	return collections_value.MakeValueSetFromSlice(CheckedOutKeyerObjectId)
}

func MakeCheckedOutMutableSet() CheckedOutMutableSet {
	return collections_value.MakeMutableValueSet(CheckedOutKeyerObjectId)
}
