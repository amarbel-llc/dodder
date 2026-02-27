package sku

// type (
// 	SkuType           = ExternalLike
// 	SkuTypeSet        = ExternalLikeSet
// 	SkuTypeSetMutable = ExternalLikeMutableSet
// 	ObjectFactory     = objectFactoryExternalLike
// )

// var (
// 	MakeSkuType                = makeExternalLike
// 	CloneSkuType               = cloneExternalLike
// 	CloneSkuTypeFromTransacted = cloneFromTransactedExternalLike
// 	MakeSkuTypeSetMutable      = MakeExternalLikeMutableSet
// )

type (
	// SkuType is a remnant of a refactoring where there used to be several
	// different structure to represent checked out objects.
	SkuType           = *CheckedOut
	SkuTypeSet        = CheckedOutSet
	SkuTypeSetMutable = CheckedOutMutableSet
	ObjectFactory     = objectFactoryCheckedOut
)

var (
	MakeSkuType                = makeCheckedOut
	CloneSkuType               = cloneCheckedOut
	CloneSkuTypeFromTransacted = cloneFromTransactedCheckedOut
	MakeSkuTypeSetMutable      = MakeCheckedOutMutableSet
)
