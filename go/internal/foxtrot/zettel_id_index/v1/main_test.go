package zettel_id_index

import (
	"testing"

	"code.linenisgreat.com/dodder/go/internal/_/coordinates"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"code.linenisgreat.com/dodder/go/lib/delta/collections"
)

// validCoordinateIds computes the set of valid coordinate IDs for a
// word list of size (lMax+1) x (rMax+1), matching v0's nested loop.
func validCoordinateIds(lMax, rMax int) map[int]bool {
	ids := make(map[int]bool)
	for l := 0; l <= lMax; l++ {
		for r := 0; r <= rMax; r++ {
			k := coordinates.ZettelIdCoordinate{
				Left:  coordinates.Int(l),
				Right: coordinates.Int(r),
			}
			ids[int(k.Id())] = true
		}
	}
	return ids
}

// maxCoordinateId returns the largest coordinate ID for the given bounds.
func maxCoordinateId(lMax, rMax int) int {
	maxId := 0
	for l := 0; l <= lMax; l++ {
		for r := 0; r <= rMax; r++ {
			k := coordinates.ZettelIdCoordinate{
				Left:  coordinates.Int(l),
				Right: coordinates.Int(r),
			}
			id := int(k.Id())
			if id > maxId {
				maxId = id
			}
		}
	}
	return maxId
}

func TestResetBitsetSizeTooSmall(t1 *testing.T) {
	t := ui.T{T: t1}

	// Simulate small word lists: 4 left words, 3 right words
	lMax := 3
	rMax := 2

	// v1 currently computes: lMax * rMax = 6 bits
	v1Size := lMax * rMax

	// The actual max coordinate ID for (3, 2):
	// (3, 2) → level 6, Id = Extrema(6).Left + 3 = 16 + 3 = 19
	actualMaxId := maxCoordinateId(lMax, rMax)

	if v1Size >= actualMaxId {
		t.Fatalf(
			"expected v1 bitset size %d to be smaller than max coordinate ID %d",
			v1Size,
			actualMaxId,
		)
	}

	t.Logf(
		"v1 allocates %d bits but max coordinate ID is %d — %d IDs unreachable",
		v1Size,
		actualMaxId,
		actualMaxId-v1Size,
	)
}

func TestResetMissesValidIds(t1 *testing.T) {
	t := ui.T{T: t1}

	lMax := 3
	rMax := 2

	// What v1 currently does: all bits ON in a bitset of size lMax*rMax
	v1Bitset := collections.MakeBitsetOn(lMax * rMax)

	// What v0 does: compute all valid coordinate IDs
	validIds := validCoordinateIds(lMax, rMax)

	// Count how many valid IDs v1's bitset can't even represent via Get()
	missing := 0
	for id := range validIds {
		if !v1Bitset.Get(id) {
			missing++
		}
	}

	if missing == 0 {
		t.Fatalf("expected some valid coordinate IDs to be missing from v1's bitset")
	}

	t.Logf(
		"v1 bitset is missing %d of %d valid coordinate IDs",
		missing,
		len(validIds),
	)
}

func TestResetIncludesInvalidIds(t1 *testing.T) {
	t := ui.T{T: t1}

	lMax := 3
	rMax := 2

	// What v1 currently does: sequential bits 0..5 are ON
	v1Bitset := collections.MakeBitsetOn(lMax * rMax)

	// What v0 does: specific coordinate IDs are valid
	validIds := validCoordinateIds(lMax, rMax)

	// Check how many ON bits in v1 correspond to invalid coordinate IDs
	invalid := 0
	v1Bitset.EachOn(func(bit int) error {
		if !validIds[bit] {
			invalid++
		}
		return nil
	})

	if invalid == 0 {
		t.Fatalf("expected some v1 bitset ON bits to not be valid coordinate IDs")
	}

	t.Logf(
		"v1 bitset has %d ON bits that are NOT valid coordinate IDs",
		invalid,
	)
}

func TestCoordinateIdsAreNotSequential(t1 *testing.T) {
	t := ui.T{T: t1}

	// Prove that coordinate IDs are not 0, 1, 2, ... N-1
	// The triangular mapping produces gaps
	lMax := 3
	rMax := 2

	validIds := validCoordinateIds(lMax, rMax)
	totalValid := len(validIds)

	// If IDs were sequential 0..N-1, all would be < totalValid
	outOfSequentialRange := 0
	for id := range validIds {
		if id >= totalValid {
			outOfSequentialRange++
		}
	}

	if outOfSequentialRange == 0 {
		t.Fatalf("expected coordinate IDs to NOT be sequential 0..N-1")
	}

	t.Logf(
		"%d of %d valid coordinate IDs are >= %d, proving non-sequential mapping",
		outOfSequentialRange,
		totalValid,
		totalValid,
	)
}
