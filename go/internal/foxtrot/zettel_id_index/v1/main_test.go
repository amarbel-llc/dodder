package zettel_id_index

import (
	"encoding"
	"math/rand"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/0/coordinates"
	"code.linenisgreat.com/dodder/go/lib/bravo/collections"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
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
	t := ui.MakeT(t1)

	lMax := 3
	rMax := 2

	oldSize := lMax * rMax

	maxCoord := coordinates.ZettelIdCoordinate{
		Left:  coordinates.Int(lMax),
		Right: coordinates.Int(rMax),
	}
	actualMaxId := int(maxCoord.Id())

	t.AssertTrue(oldSize < actualMaxId, "expected old bitset size to be smaller than max coordinate ID")

	t.Logf(
		"old approach allocates %d bits but max coordinate ID is %d",
		oldSize,
		actualMaxId,
	)
}

func TestOldResetMissesValidIds(t1 *testing.T) {
	t := ui.MakeT(t1)

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

	t.AssertFalse(missing == 0, "expected some valid coordinate IDs to be missing from old bitset")

	t.Logf("old bitset is missing %d of %d valid coordinate IDs", missing, len(validIds))
}

func TestCoordinateIdsAreNotSequential(t1 *testing.T) {
	t := ui.MakeT(t1)

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

	t.AssertFalse(outOfSequentialRange == 0, "expected coordinate IDs to NOT be sequential 0..N-1")

	t.Logf(
		"%d of %d valid coordinate IDs are >= %d, proving non-sequential mapping",
		outOfSequentialRange,
		totalValid,
		totalValid,
	)
}

// --- Correctness tests: verify the fixed coordinate-aware Reset ---

func TestFixedResetContainsAllValidIds(t1 *testing.T) {
	t := ui.MakeT(t1)

	lMax := 3
	rMax := 2

	bs := makeBitsetFromCoordinates(lMax, rMax)
	validIds := validCoordinateIds(lMax, rMax)

	for id := range validIds {
		if !bs.Get(id) {
			t.Errorf("valid coordinate ID %d is not set in bitset", id)
		}
	}

	t.AssertEqual(len(validIds), bs.CountOn())
}

func TestFixedResetContainsNoInvalidIds(t1 *testing.T) {
	t := ui.MakeT(t1)

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
	t := ui.MakeT(t1)

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

	t.AssertEqual(len(validIds), bs.CountOn())
}

// Probe for take4's "zettel ids exhausted" (dodder #16): allocating ~519 ids
// from a live-sized pool (184 x 230) after reserving a few thousand used ids
// must never fail while CountOn() > 0. Mirrors CreateZettelId's selection
// (rand.Intn(CountOn()) then NthOn), once on a fresh bitset and once after a
// marshal/unmarshal round trip into the SAME bitset value (the index's
// Reset -> Flush -> readIfNecessary sequence).
func TestAllocationAfterReserveAtLiveScale(t1 *testing.T) {
	t := ui.MakeT(t1)

	lMax, rMax := 183, 229

	allocate := func(bs collections.Bitset, label string) {
		reserved := 0

		for id := 0; reserved < 5000 && id < bs.Len(); id += 7 {
			if bs.Get(id) {
				bs.DelIfPresent(id)
				reserved++
			}
		}

		real := 0
		bs.EachOn(func(int) error { real++; return nil })

		t.AssertEqual(real, bs.CountOn())

		for i := 0; i < 519; i++ {
			count := bs.CountOn()
			t.AssertTrue(count > 0, label+": pool unexpectedly empty")

			n, ok := bs.NthOn(rand.Intn(count))
			if !ok {
				t.Fatalf("%s: NthOn failed at allocation %d (CountOn=%d)", label, i, count)
			}

			bs.DelIfPresent(n)
		}
	}

	allocate(makeBitsetFromCoordinates(lMax, rMax), "fresh")

	// Round trip: unmarshal the flushed bytes back INTO the populated bitset,
	// as readIfNecessary does with index.bitset after Reset populated it.
	populated := makeBitsetFromCoordinates(lMax, rMax)
	encoded, err := populated.(encoding.BinaryMarshaler).MarshalBinary()
	t.AssertNoError(err)
	t.AssertNoError(populated.(encoding.BinaryUnmarshaler).UnmarshalBinary(encoded))

	allocate(populated, "round-trip")
}

func TestFixedResetRealisticSize(t1 *testing.T) {
	t := ui.MakeT(t1)

	// Realistic word list sizes (dodder ships ~100 left, ~50 right words)
	lMax := 99
	rMax := 49

	bs := makeBitsetFromCoordinates(lMax, rMax)
	expectedCount := (lMax + 1) * (rMax + 1)

	t.AssertEqual(expectedCount, bs.CountOn())

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
