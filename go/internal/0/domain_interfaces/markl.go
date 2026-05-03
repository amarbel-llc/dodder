package domain_interfaces

import (
	madder_di "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"

	// Side-effect blank import: triggers init() in madder's
	// markl_registrations, which calls markl.RegisterPurpose for every
	// purpose dodder uses (Repo*, Object*, Blob*, Madder*, Request*) plus
	// the dodder/zit private-key aliases. Before the markl-fork drop,
	// dodder's own internal/bravo/markl/purposes.go ran these
	// registrations via init() and any code that imported markl got them
	// transitively. After the drop, dodder consumes madder/pkgs/markl
	// (types only, no init), so the registrations have to be triggered
	// explicitly. This package is the lowest dodder-internal tier that
	// every downstream caller transitively imports — the most uniform
	// place to keep production binaries and unit-test binaries in sync.
	_ "github.com/amarbel-llc/madder/go/pkgs/markl_registrations"
)

// These interfaces are byte-identical between dodder and madder. Aliased
// from madder's public domain_interfaces facade so dodder ships a single
// source of truth without forcing every importer to re-spell its imports.
type (
	MarklFormat                                                         = madder_di.MarklFormat
	FormatHash                                                          = madder_di.FormatHash
	MarklFormatGetter                                                   = madder_di.MarklFormatGetter
	Hash                                                                = madder_di.Hash
	MarklId                                                             = madder_di.MarklId
	MarklIdMutable                                                      = madder_di.MarklIdMutable
	MarklIdGetter                                                       = madder_di.MarklIdGetter
	DigestWriteMap                                                      = madder_di.DigestWriteMap
	Lock[KEY interfaces.Value, KEY_PTR interfaces.ValuePtr[KEY]]        = madder_di.Lock[KEY, KEY_PTR]
	LockMutable[KEY interfaces.Value, KEY_PTR interfaces.ValuePtr[KEY]] = madder_di.LockMutable[KEY, KEY_PTR]
)
