package commands_dodder

import "code.linenisgreat.com/dodder/go/internal/foxtrot/sku"

// inventoryListUnionDeduper collapses exact (object-id, tai, blob-digest)
// duplicates across an inventory-list union to a single object, while keeping
// every distinct triple.
//
// This is LOAD-BEARING, not an optimization (dodder#392). import_plan's builder
// does NOT collapse exact duplicates for a union: MakeLocalBuilder configures no
// content dedup, and the builder's within-batch path reassigns a fresh tai to
// any second object sharing an (object-id, tai) UNCONDITIONALLY — it never
// compares digests (see import_plan/builder.go AddObject). So an exact duplicate
// that reached the builder would be committed as a spurious extra revision (a
// real 44k-object import produced hundreds of these). Exact duplicates must be
// dropped here, before the builder ever sees them.
//
// Same-(id,tai)-different-digest COLLISIONS and same-id-different-tai VERSIONS
// are deliberately KEPT: the former are the genuine collisions the builder's
// reassign exists for, and the latter are ordinary history a consolidation must
// preserve. The script therefore sees the whole merged graph, exactly once per
// distinct triple.
type inventoryListUnionDeduper struct {
	seen map[string]struct{}
}

func makeInventoryListUnionDeduper() inventoryListUnionDeduper {
	return inventoryListUnionDeduper{seen: make(map[string]struct{})}
}

// keep reports whether object is the first occurrence of its exact
// (object-id, tai, blob-digest) triple, recording it. An exact duplicate
// returns false (collapse); every distinct triple returns true.
func (d inventoryListUnionDeduper) keep(object *sku.Transacted) bool {
	key := object.GetObjectId().String() + "\x00" +
		object.GetTai().String() + "\x00" +
		object.GetBlobDigest().String()

	if _, seen := d.seen[key]; seen {
		return false
	}

	d.seen[key] = struct{}{}

	return true
}
