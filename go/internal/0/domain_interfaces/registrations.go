package domain_interfaces

// Side-effect blank import: triggers init() in madder's
// markl_registrations, which calls markl.RegisterPurpose for every
// purpose dodder uses. After the markl-fork drop, dodder consumes
// madder/pkgs/markl (types only, no init), so the registrations
// have to be triggered explicitly. This package is the lowest
// dodder-internal tier that every downstream caller transitively
// imports — the most uniform place to keep production binaries
// and unit-test binaries in sync.
import _ "github.com/amarbel-llc/madder/go/pkgs/markl_registrations"
