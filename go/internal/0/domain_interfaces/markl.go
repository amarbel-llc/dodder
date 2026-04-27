package domain_interfaces

import (
	madder_di "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

// These interfaces are byte-identical between dodder and madder. Aliased
// from madder's public domain_interfaces facade so dodder ships a single
// source of truth without forcing every importer to re-spell its imports.
type (
	MarklFormat       = madder_di.MarklFormat
	FormatHash        = madder_di.FormatHash
	MarklFormatGetter = madder_di.MarklFormatGetter
	Hash              = madder_di.Hash
	MarklId           = madder_di.MarklId
	MarklIdMutable    = madder_di.MarklIdMutable
	MarklIdGetter     = madder_di.MarklIdGetter
	DigestWriteMap    = madder_di.DigestWriteMap
)

// Lock and LockMutable are not yet exported from madder's pkgs/
// domain_interfaces facade, so dodder still defines them locally. Once
// madder exposes them, swap to type aliases as above.
type (
	Lock[
		KEY interfaces.Value,
		KEY_PTR interfaces.ValuePtr[KEY],
	] interface {
		GetKey() KEY
		GetValue() MarklId
		IsEmpty() bool
	}

	LockMutable[
		KEY interfaces.Value,
		KEY_PTR interfaces.ValuePtr[KEY],
	] interface {
		Lock[KEY, KEY_PTR]
		GetKeyMutable() KEY_PTR
		GetValueMutable() MarklIdMutable
	}
)
