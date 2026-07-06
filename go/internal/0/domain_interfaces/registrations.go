package domain_interfaces

// Side-effect blank import: triggers init() in dodder's own
// markl_registrations, which calls markl.RegisterPurpose for every
// dodder-* purpose and transitively activates madder's (madder-*
// purposes + legacy private-key aliases), piggy's core format +
// piggy-* registrations, and the real age/agent implementations.
// dodder used to activate madder's markl_registrations directly here;
// the madder#255 ownership cutover moved the dodder-* vocabulary into
// dodder's own package, which now chains the rest. This package is the
// lowest dodder-internal tier that every downstream caller
// transitively imports — the most uniform place to keep production
// binaries and unit-test binaries in sync.
import _ "code.linenisgreat.com/dodder/go/internal/0/markl_registrations"
