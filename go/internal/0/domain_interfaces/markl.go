package domain_interfaces

import (
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
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
	MarklFormat                                                         = mad_domain_interfaces.MarklFormat
	FormatHash                                                          = mad_domain_interfaces.FormatHash
	MarklFormatGetter                                                   = mad_domain_interfaces.MarklFormatGetter
	Hash                                                                = mad_domain_interfaces.Hash
	MarklId                                                             = mad_domain_interfaces.MarklId
	MarklIdMutable                                                      = mad_domain_interfaces.MarklIdMutable
	MarklIdGetter                                                       = mad_domain_interfaces.MarklIdGetter
	DigestWriteMap                                                      = mad_domain_interfaces.DigestWriteMap
	Lock[KEY interfaces.Value, KEY_PTR interfaces.ValuePtr[KEY]]        = mad_domain_interfaces.Lock[KEY, KEY_PTR]
	LockMutable[KEY interfaces.Value, KEY_PTR interfaces.ValuePtr[KEY]] = mad_domain_interfaces.LockMutable[KEY, KEY_PTR]
)
