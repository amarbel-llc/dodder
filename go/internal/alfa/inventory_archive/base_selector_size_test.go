//go:build test && debug

package inventory_archive

import (
	"testing"
)

type testBlobSet struct {
	blobs []BlobMetadata
}

func (s *testBlobSet) Len() int                  { return len(s.blobs) }
func (s *testBlobSet) At(index int) BlobMetadata { return s.blobs[index] }

type testAssignments struct {
	assignments map[int]int
	errors      map[int]error
}

func newTestAssignments() *testAssignments {
	return &testAssignments{
		assignments: make(map[int]int),
		errors:      make(map[int]error),
	}
}

func (a *testAssignments) Assign(blobIndex, baseIndex int) {
	a.assignments[blobIndex] = baseIndex
}

func (a *testAssignments) AssignError(blobIndex int, err error) {
	a.errors[blobIndex] = err
}

func TestSizeBasedSelectorGroupsSimilarSizes(t *testing.T) {
	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Id: nil, Size: 1000},
			{Id: nil, Size: 1100},
			{Id: nil, Size: 1200},
			{Id: nil, Size: 5000},
			{Id: nil, Size: 5500},
		},
	}

	selector := &SizeBasedSelector{
		MinBlobSize: 100,
		MaxBlobSize: 10000,
		SizeRatio:   2.0,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	// Blobs 0, 1, 2 should be grouped (within 2x ratio of each other)
	// Blobs 3, 4 should be grouped (within 2x ratio)
	// The largest in each group is the base
	if len(assignments.assignments) == 0 {
		t.Fatal("expected some delta assignments")
	}

	// Verify no self-assignments
	for blobIdx, baseIdx := range assignments.assignments {
		if blobIdx == baseIdx {
			t.Errorf("blob %d assigned to itself", blobIdx)
		}
	}

	// Blob 2 (size 1200) should be the base for the first group
	// So blobs 0 and 1 should be assigned to blob 2
	if base, ok := assignments.assignments[0]; !ok || base != 2 {
		t.Errorf("blob 0: expected base 2, got %d (ok=%v)", base, ok)
	}

	if base, ok := assignments.assignments[1]; !ok || base != 2 {
		t.Errorf("blob 1: expected base 2, got %d (ok=%v)", base, ok)
	}

	// Blob 2 should NOT be in assignments (it's a base)
	if _, ok := assignments.assignments[2]; ok {
		t.Error("blob 2 should be a base, not assigned")
	}

	// Blob 4 (size 5500) should be the base for the second group
	if base, ok := assignments.assignments[3]; !ok || base != 4 {
		t.Errorf("blob 3: expected base 4, got %d (ok=%v)", base, ok)
	}

	if _, ok := assignments.assignments[4]; ok {
		t.Error("blob 4 should be a base, not assigned")
	}
}

func TestSizeBasedSelectorSkipsSmallBlobs(t *testing.T) {
	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Id: nil, Size: 50},
			{Id: nil, Size: 60},
		},
	}

	selector := &SizeBasedSelector{
		MinBlobSize: 100,
		MaxBlobSize: 10000,
		SizeRatio:   2.0,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	if len(assignments.assignments) != 0 {
		t.Errorf("expected no assignments for small blobs, got %d",
			len(assignments.assignments))
	}
}

func TestSizeBasedSelectorSkipsLargeBlobs(t *testing.T) {
	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Id: nil, Size: 20000000},
			{Id: nil, Size: 20000001},
		},
	}

	selector := &SizeBasedSelector{
		MinBlobSize: 100,
		MaxBlobSize: 10000000,
		SizeRatio:   2.0,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	if len(assignments.assignments) != 0 {
		t.Errorf("expected no assignments for large blobs, got %d",
			len(assignments.assignments))
	}
}

func TestSizeBasedSelectorSingleBlob(t *testing.T) {
	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Id: nil, Size: 1000},
		},
	}

	selector := &SizeBasedSelector{
		MinBlobSize: 100,
		MaxBlobSize: 10000,
		SizeRatio:   2.0,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	if len(assignments.assignments) != 0 {
		t.Errorf("expected no assignments for single blob, got %d",
			len(assignments.assignments))
	}
}

func TestSizeBasedSelectorEmptyBlobSet(t *testing.T) {
	blobs := &testBlobSet{blobs: nil}

	selector := &SizeBasedSelector{
		MinBlobSize: 100,
		MaxBlobSize: 10000,
		SizeRatio:   2.0,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	if len(assignments.assignments) != 0 {
		t.Errorf("expected no assignments for empty blob set, got %d",
			len(assignments.assignments))
	}
}

func TestSizeBasedSelectorDisjointGroups(t *testing.T) {
	// Blobs with sizes that are far apart (> 2x ratio) should not be grouped
	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Id: nil, Size: 100},
			{Id: nil, Size: 500},
			{Id: nil, Size: 2000},
		},
	}

	selector := &SizeBasedSelector{
		MinBlobSize: 50,
		MaxBlobSize: 10000,
		SizeRatio:   2.0,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	// Each blob is in its own group (100 -> 500 is 5x, 500 -> 2000 is 4x)
	// No groups have 2+ members
	if len(assignments.assignments) != 0 {
		t.Errorf("expected no assignments for disjoint blobs, got %d",
			len(assignments.assignments))
	}
}
