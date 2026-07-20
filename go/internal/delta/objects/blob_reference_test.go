package objects

import (
	"testing"

	"code.linenisgreat.com/dodder/go/lib/0/collections_slice"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

func TestBlobReferencesAddSortsByKey(t1 *testing.T) {
	t := ui.MakeT(t1)
	var refs BlobReferences

	// Create three markl.Id values with distinct blech32 encodings.
	// We add them in reverse order to verify sorting.
	marklIds := makeThreeMarklIds(&t)

	// Add in reverse order: last, middle, first
	refs.Add(marklIds[2], markl.Lock[SeqId, *SeqId]{})
	refs.Add(marklIds[1], markl.Lock[SeqId, *SeqId]{})
	refs.Add(marklIds[0], markl.Lock[SeqId, *SeqId]{})

	// Collect results
	var got []string
	for id := range refs.All() {
		got = append(got, id.String())
	}

	t.AssertLen(3, got, "blob reference entries")

	for i := 1; i < len(got); i++ {
		if got[i-1] >= got[i] {
			t.Errorf(
				"blob references not sorted: got[%d]=%q >= got[%d]=%q",
				i-1, got[i-1], i, got[i],
			)
		}
	}
}

func makeThreeMarklIds(t *ui.T) [3]markl.Id {
	t.Helper()

	format, err := markl.GetFormatOrError("blake2b256")
	t.AssertNoError(err)

	size := format.GetSize()
	var result [3]markl.Id

	for i := range result {
		data := make([]byte, size)
		// Fill with different byte values to get distinct blech32 encodings
		for j := range data {
			data[j] = byte((i + 1) * 50) // 50, 100, 150
		}
		err := result[i].SetMarklId("blake2b256", data)
		t.AssertNoError(err)
	}

	// Sort by string to establish known order
	ordered := collections_slice.Slice[markl.Id](result[:])
	ordered.SortByStringFunc(func(id markl.Id) string { return id.String() })
	copy(result[:], ordered)

	return result
}
