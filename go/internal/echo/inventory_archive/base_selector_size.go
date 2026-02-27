package inventory_archive

import (
	"sort"
)

// SizeBasedSelector groups blobs by similar size and assigns deltas within
// each group against the largest blob as the base.
//
// TODO: Content-type base selection strategy — madder queries dodder for
// blob type info (binary flag), groups text blobs separately from binary.
//
// TODO: Object-history base selection strategy — dodder provides
// related-object hash chains, packer deltas successive versions of the
// same object against each other.
type SizeBasedSelector struct {
	MinBlobSize uint64
	MaxBlobSize uint64
	SizeRatio   float64
}

var _ BaseSelector = &SizeBasedSelector{}

func (s *SizeBasedSelector) SelectBases(
	blobs BlobSet,
	assignments DeltaAssignments,
) {
	n := blobs.Len()
	if n < 2 {
		return
	}

	type indexedBlob struct {
		originalIndex int
		size          uint64
	}

	sorted := make([]indexedBlob, 0, n)

	for i := range n {
		meta := blobs.At(i)

		if meta.Size < s.MinBlobSize || meta.Size > s.MaxBlobSize {
			continue
		}

		sorted = append(sorted, indexedBlob{
			originalIndex: i,
			size:          meta.Size,
		})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].size < sorted[j].size
	})

	if len(sorted) < 2 {
		return
	}

	groupStart := 0
	for i := 1; i <= len(sorted); i++ {
		inGroup := i < len(sorted) &&
			float64(sorted[i].size) <= float64(sorted[groupStart].size)*s.SizeRatio

		if inGroup {
			continue
		}

		// End of group: [groupStart, i)
		if i-groupStart >= 2 {
			// Largest blob in group is the base (last in sorted order)
			baseIdx := sorted[i-1].originalIndex

			for j := groupStart; j < i-1; j++ {
				assignments.Assign(sorted[j].originalIndex, baseIdx)
			}
		}

		groupStart = i
	}
}
