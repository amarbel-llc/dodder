package zettel_id_index

import (
	"testing"

	"code.linenisgreat.com/dodder/go/internal/0/coordinates"
	"code.linenisgreat.com/dodder/go/lib/bravo/collections"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

const (
	benchLMax = 99
	benchRMax = 49
)

// --- Reset benchmarks: building the initial available ID set ---

func BenchmarkResetV0Map(b *testing.B) {
	for range b.N {
		m := make(map[int]bool, benchLMax*benchRMax)
		for l := 0; l <= benchLMax; l++ {
			for r := 0; r <= benchRMax; r++ {
				k := coordinates.ZettelIdCoordinate{
					Left:  coordinates.Int(l),
					Right: coordinates.Int(r),
				}
				m[int(k.Id())] = true
			}
		}
	}
}

func BenchmarkResetV1Bitset(b *testing.B) {
	for range b.N {
		maxCoord := coordinates.ZettelIdCoordinate{
			Left:  coordinates.Int(benchLMax),
			Right: coordinates.Int(benchRMax),
		}
		bs := collections.MakeBitset(int(maxCoord.Id()) + 1)
		for l := 0; l <= benchLMax; l++ {
			for r := 0; r <= benchRMax; r++ {
				k := coordinates.ZettelIdCoordinate{
					Left:  coordinates.Int(l),
					Right: coordinates.Int(r),
				}
				bs.Add(int(k.Id()))
			}
		}
	}
}

// --- AddZettelId benchmarks: marking an ID as used ---

func BenchmarkAddZettelIdV0Map(b *testing.B) {
	m := make(map[int]bool, benchLMax*benchRMax)
	for l := 0; l <= benchLMax; l++ {
		for r := 0; r <= benchRMax; r++ {
			k := coordinates.ZettelIdCoordinate{
				Left:  coordinates.Int(l),
				Right: coordinates.Int(r),
			}
			m[int(k.Id())] = true
		}
	}

	ids := make([]int, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}

	b.ResetTimer()
	for i := range b.N {
		delete(m, ids[i%len(ids)])
	}
}

func BenchmarkAddZettelIdV1Bitset(b *testing.B) {
	maxCoord := coordinates.ZettelIdCoordinate{
		Left:  coordinates.Int(benchLMax),
		Right: coordinates.Int(benchRMax),
	}
	bs := collections.MakeBitset(int(maxCoord.Id()) + 1)
	for l := 0; l <= benchLMax; l++ {
		for r := 0; r <= benchRMax; r++ {
			k := coordinates.ZettelIdCoordinate{
				Left:  coordinates.Int(l),
				Right: coordinates.Int(r),
			}
			bs.Add(int(k.Id()))
		}
	}

	ids := make([]int, 0)
	bs.EachOn(func(id int) error {
		ids = append(ids, id)
		return nil
	})

	b.ResetTimer()
	for i := range b.N {
		bs.DelIfPresent(ids[i%len(ids)])
	}
}

// --- CreateZettelId benchmarks: finding an available ID ---

func BenchmarkCreateZettelIdV0MapIterate(b *testing.B) {
	m := make(map[int]bool, benchLMax*benchRMax)
	for l := 0; l <= benchLMax; l++ {
		for r := 0; r <= benchRMax; r++ {
			k := coordinates.ZettelIdCoordinate{
				Left:  coordinates.Int(l),
				Right: coordinates.Int(r),
			}
			m[int(k.Id())] = true
		}
	}

	b.ResetTimer()
	for range b.N {
		// v0 iterates map to pick an element
		for n := range m {
			_ = n
			break
		}
	}
}

func BenchmarkCreateZettelIdV1BitsetNthOn(b *testing.B) {
	maxCoord := coordinates.ZettelIdCoordinate{
		Left:  coordinates.Int(benchLMax),
		Right: coordinates.Int(benchRMax),
	}
	bs := collections.MakeBitset(int(maxCoord.Id()) + 1)
	for l := 0; l <= benchLMax; l++ {
		for r := 0; r <= benchRMax; r++ {
			k := coordinates.ZettelIdCoordinate{
				Left:  coordinates.Int(l),
				Right: coordinates.Int(r),
			}
			bs.Add(int(k.Id()))
		}
	}

	target := bs.CountOn() / 2

	b.ResetTimer()
	for range b.N {
		bs.NthOn(target)
	}
}

// --- Memory: report allocation sizes ---

func TestMemoryComparison(t1 *testing.T) {
	t := ui.MakeT(t1)

	// v0: map[int]bool
	m := make(map[int]bool, benchLMax*benchRMax)
	for l := 0; l <= benchLMax; l++ {
		for r := 0; r <= benchRMax; r++ {
			k := coordinates.ZettelIdCoordinate{
				Left:  coordinates.Int(l),
				Right: coordinates.Int(r),
			}
			m[int(k.Id())] = true
		}
	}

	// v1: bitset
	maxCoord := coordinates.ZettelIdCoordinate{
		Left:  coordinates.Int(benchLMax),
		Right: coordinates.Int(benchRMax),
	}
	bs := collections.MakeBitset(int(maxCoord.Id()) + 1)
	for l := 0; l <= benchLMax; l++ {
		for r := 0; r <= benchRMax; r++ {
			k := coordinates.ZettelIdCoordinate{
				Left:  coordinates.Int(l),
				Right: coordinates.Int(r),
			}
			bs.Add(int(k.Id()))
		}
	}

	v0Entries := len(m)
	// map[int]bool: ~8 bytes key + 1 byte value + ~24 bytes overhead per entry
	v0EstimatedBytes := v0Entries * 33

	// bitset: capacity in bits / 8
	v1Bytes := bs.Len() / 8

	t.Logf("v0 map: %d entries, ~%d bytes estimated", v0Entries, v0EstimatedBytes)
	t.Logf("v1 bitset: %d bits (%d bytes), %d IDs tracked", bs.Len(), v1Bytes, bs.CountOn())
	t.Logf("v1 is ~%.0fx smaller", float64(v0EstimatedBytes)/float64(v1Bytes))
}
