package zettel_id_index

import (
	"testing"

	"code.linenisgreat.com/dodder/go/internal/0/coordinates"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"code.linenisgreat.com/dodder/go/lib/bravo/collections"
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

// makeBitsetFromCoordinates builds a bitset the same way Reset() should.
func makeBitsetFromCoordinates(lMax, rMax int) collections.Bitset {
	maxCoord := coordinates.ZettelIdCoordinate{
		Left:  coordinates.Int(lMax),
		Right: coordinates.Int(rMax),
	}
	bs := collections.MakeBitset(int(maxCoord.Id()) + 1)

	for l := 0; l <= lMax; l++ {
		for r := 0; r <= rMax; r++ {
			k := coordinates.ZettelIdCoordinate{
				Left:  coordinates.Int(l),
				Right: coordinates.Int(r),
			}
			bs.Add(int(k.Id()))
		}
	}

	return bs
}

// --- Regression tests: prove the OLD MakeBitsetOn approach was wrong ---

func TestOldResetBitsetSizeTooSmall(t1 *testing.T) {
	t := ui.T{T: t1}

	lMax := 3
	rMax := 2

	oldSize := lMax * rMax

	maxCoord := coordinates.ZettelIdCoordinate{
		Left:  coordinates.Int(lMax),
		Right: coordinates.Int(rMax),
	}
	actualMaxId := int(maxCoord.Id())

	if oldSize >= actualMaxId {
		t.Fatalf(
			"expected old bitset size %d to be smaller than max coordinate ID %d",
			oldSize,
			actualMaxId,
		)
	}

	t.Logf(
		"old approach allocates %d bits but max coordinate ID is %d",
		oldSize,
		actualMaxId,
	)
}

func TestOldResetMissesValidIds(t1 *testing.T) {
	t := ui.T{T: t1}

	lMax := 3
	rMax := 2

	oldBitset := collections.MakeBitsetOn(lMax * rMax)
	validIds := validCoordinateIds(lMax, rMax)

	missing := 0
	for id := range validIds {
		if !oldBitset.Get(id) {
			missing++
		}
	}

	if missing == 0 {
		t.Fatalf("expected some valid coordinate IDs to be missing from old bitset")
	}

	t.Logf("old bitset is missing %d of %d valid coordinate IDs", missing, len(validIds))
}

func TestCoordinateIdsAreNotSequential(t1 *testing.T) {
	t := ui.T{T: t1}

	lMax := 3
	rMax := 2

	validIds := validCoordinateIds(lMax, rMax)
	totalValid := len(validIds)

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

// --- Correctness tests: verify the fixed coordinate-aware Reset ---

func TestFixedResetContainsAllValidIds(t1 *testing.T) {
	t := ui.T{T: t1}

	lMax := 3
	rMax := 2

	bs := makeBitsetFromCoordinates(lMax, rMax)
	validIds := validCoordinateIds(lMax, rMax)

	for id := range validIds {
		if !bs.Get(id) {
			t.Errorf("valid coordinate ID %d is not set in bitset", id)
		}
	}

	if bs.CountOn() != len(validIds) {
		t.Errorf(
			"expected CountOn=%d but got %d",
			len(validIds),
			bs.CountOn(),
		)
	}
}

func TestFixedResetContainsNoInvalidIds(t1 *testing.T) {
	t := ui.T{T: t1}

	lMax := 3
	rMax := 2

	bs := makeBitsetFromCoordinates(lMax, rMax)
	validIds := validCoordinateIds(lMax, rMax)

	invalid := 0
	bs.EachOn(func(bit int) error {
		if !validIds[bit] {
			invalid++
			t.Errorf("bit %d is ON but is not a valid coordinate ID", bit)
		}
		return nil
	})

	if invalid > 0 {
		t.Errorf("%d ON bits are not valid coordinate IDs", invalid)
	}
}

func TestFixedResetRoundTripCoordinates(t1 *testing.T) {
	t := ui.T{T: t1}

	lMax := 5
	rMax := 4

	bs := makeBitsetFromCoordinates(lMax, rMax)
	validIds := validCoordinateIds(lMax, rMax)

	// Every ON bit should round-trip back to valid (l, r) coordinates
	bs.EachOn(func(id int) error {
		k := &coordinates.ZettelIdCoordinate{}
		k.SetInt(coordinates.Int(id))

		if int(k.Left) > lMax || int(k.Right) > rMax {
			t.Errorf(
				"ID %d maps to (%d, %d) which is outside bounds (%d, %d)",
				id, k.Left, k.Right, lMax, rMax,
			)
		}

		roundTripped := int(k.Id())
		if roundTripped != id {
			t.Errorf("ID %d round-trips to %d", id, roundTripped)
		}

		return nil
	})

	if bs.CountOn() != len(validIds) {
		t.Errorf("expected %d ON bits but got %d", len(validIds), bs.CountOn())
	}
}

func TestFixedResetRealisticSize(t1 *testing.T) {
	t := ui.T{T: t1}

	// Realistic word list sizes (dodder ships ~100 left, ~50 right words)
	lMax := 99
	rMax := 49

	bs := makeBitsetFromCoordinates(lMax, rMax)
	expectedCount := (lMax + 1) * (rMax + 1)

	if bs.CountOn() != expectedCount {
		t.Errorf("expected %d available IDs but got %d", expectedCount, bs.CountOn())
	}

	maxCoord := coordinates.ZettelIdCoordinate{
		Left:  coordinates.Int(lMax),
		Right: coordinates.Int(rMax),
	}

	t.Logf(
		"realistic bitset: %d available IDs, max coordinate ID %d, bitset capacity %d bits",
		bs.CountOn(),
		maxCoord.Id(),
		bs.Len(),
	)
}
