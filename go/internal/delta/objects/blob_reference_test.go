package objects

import (
	"testing"

	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/lib/bravo/collections_slice"
)

func TestBlobReferencesAddSortsByKey(t *testing.T) {
	var refs BlobReferences

	// Create three markl.Id values with distinct blech32 encodings.
	// We add them in reverse order to verify sorting.
	marklIds := makeThreeMarklIds(t)

	// Add in reverse order: last, middle, first
	refs.Add(marklIds[2], markl.Lock[SeqId, *SeqId]{})
	refs.Add(marklIds[1], markl.Lock[SeqId, *SeqId]{})
	refs.Add(marklIds[0], markl.Lock[SeqId, *SeqId]{})

	// Collect results
	var got []string
	for id := range refs.All() {
		got = append(got, id.String())
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}

	for i := 1; i < len(got); i++ {
		if got[i-1] >= got[i] {
			t.Errorf(
				"blob references not sorted: got[%d]=%q >= got[%d]=%q",
				i-1, got[i-1], i, got[i],
			)
		}
	}
}

func makeThreeMarklIds(t *testing.T) [3]markl.Id {
	t.Helper()

	format, err := markl.GetFormatOrError("blake2b256")
	if err != nil {
		t.Fatalf("getting blake2b256 format: %v", err)
	}

	size := format.GetSize()
	var result [3]markl.Id

	for i := range result {
		data := make([]byte, size)
		// Fill with different byte values to get distinct blech32 encodings
		for j := range data {
			data[j] = byte((i + 1) * 50) // 50, 100, 150
		}
		if err := result[i].SetMarklId("blake2b256", data); err != nil {
			t.Fatalf("setting markl id %d: %v", i, err)
		}
	}

	// Sort by string to establish known order
	ordered := collections_slice.Slice[markl.Id](result[:])
	ordered.SortByStringFunc(func(id markl.Id) string { return id.String() })
	copy(result[:], ordered)

	return result
}
