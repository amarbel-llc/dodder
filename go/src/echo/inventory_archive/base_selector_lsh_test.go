//go:build test && debug

package inventory_archive

import (
	"bytes"
	"testing"
)

func makeTestSignatures(
	t *testing.T,
	blobs [][]byte,
	computer *GearCDCMinHashComputer,
) [][]uint32 {
	t.Helper()

	sigs := make([][]uint32, len(blobs))

	for i, data := range blobs {
		sig, err := computer.ComputeSignature(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("signature %d: %v", i, err)
		}

		sigs[i] = sig
	}

	return sigs
}

func TestLSHBandingSelectorAssignsSimilarBlobs(t *testing.T) {
	computer := &GearCDCMinHashComputer{
		AvgChunkSize: 48,
		MinChunkSize: 16,
		MaxChunkSize: 256,
		K:            64,
	}

	// Blob 0: original
	original := make([]byte, 1024)
	for i := range original {
		original[i] = byte(i * 7)
	}

	// Blob 1: small edit of original (change 50 bytes)
	edited := make([]byte, len(original))
	copy(edited, original)
	for i := 500; i < 550; i++ {
		edited[i] = byte(i * 13)
	}

	// Blob 2: completely different
	unrelated := make([]byte, 1024)
	for i := range unrelated {
		unrelated[i] = byte(i*17 + 128)
	}

	blobData := [][]byte{original, edited, unrelated}
	sigs := makeTestSignatures(t, blobData, computer)

	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Size: 1024, Signature: sigs[0]},
			{Size: 1024, Signature: sigs[1]},
			{Size: 1024, Signature: sigs[2]},
		},
	}

	selector := &LSHBandingSelector{
		Bands:       16,
		RowsPerBand: 4,
		MinBlobSize: 100,
		MaxBlobSize: 10000,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	// Blob 0 and blob 1 should be paired (similar)
	// One should be assigned to the other
	assigned0, has0 := assignments.assignments[0]
	assigned1, has1 := assignments.assignments[1]

	paired := (has0 && assigned0 == 1) || (has1 && assigned1 == 0)
	if !paired {
		t.Error("expected blob 0 and 1 to be paired as similar")
	}

	// Should not have mutual assignment (both assigned to each other)
	if has0 && has1 && assigned0 == 1 && assigned1 == 0 {
		t.Error("mutual delta cycle: blob 0 and 1 are each other's delta")
	}
}

func TestLSHBandingSelectorSkipsNilSignatures(t *testing.T) {
	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Size: 1024, Signature: nil},
			{Size: 1024, Signature: nil},
		},
	}

	selector := &LSHBandingSelector{
		Bands:       16,
		RowsPerBand: 4,
		MinBlobSize: 100,
		MaxBlobSize: 10000,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	if len(assignments.assignments) != 0 {
		t.Errorf("expected no assignments for nil signatures, got %d",
			len(assignments.assignments))
	}
}

func TestLSHBandingSelectorRespectsMinBlobSize(t *testing.T) {
	sig := make([]uint32, 64)
	for i := range sig {
		sig[i] = uint32(i)
	}

	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Size: 10, Signature: sig},
			{Size: 10, Signature: sig},
		},
	}

	selector := &LSHBandingSelector{
		Bands:       16,
		RowsPerBand: 4,
		MinBlobSize: 100,
		MaxBlobSize: 10000,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	if len(assignments.assignments) != 0 {
		t.Errorf("expected no assignments for small blobs, got %d",
			len(assignments.assignments))
	}
}

func TestLSHBandingSelectorRespectsMaxBlobSize(t *testing.T) {
	sig := make([]uint32, 64)
	for i := range sig {
		sig[i] = uint32(i)
	}

	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Size: 50000, Signature: sig},
			{Size: 50000, Signature: sig},
		},
	}

	selector := &LSHBandingSelector{
		Bands:       16,
		RowsPerBand: 4,
		MinBlobSize: 100,
		MaxBlobSize: 10000,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	if len(assignments.assignments) != 0 {
		t.Errorf("expected no assignments for large blobs, got %d",
			len(assignments.assignments))
	}
}

func TestLSHBandingSelectorNoSelfAssignment(t *testing.T) {
	computer := &GearCDCMinHashComputer{
		AvgChunkSize: 48,
		MinChunkSize: 16,
		MaxChunkSize: 256,
		K:            64,
	}

	data := make([]byte, 512)
	for i := range data {
		data[i] = byte(i * 3)
	}

	// Two identical blobs
	sigs := makeTestSignatures(t, [][]byte{data, data}, computer)

	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Size: 512, Signature: sigs[0]},
			{Size: 512, Signature: sigs[1]},
		},
	}

	selector := &LSHBandingSelector{
		Bands:       16,
		RowsPerBand: 4,
		MinBlobSize: 100,
		MaxBlobSize: 10000,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	for blobIdx, baseIdx := range assignments.assignments {
		if blobIdx == baseIdx {
			t.Errorf("blob %d assigned to itself", blobIdx)
		}
	}

	// At most one of the pair should be a delta; the other must be a base
	if len(assignments.assignments) > 1 {
		t.Errorf("expected at most one assignment for two equal blobs, got %d",
			len(assignments.assignments))
	}
}

func TestLSHBandingSelectorSingleBlob(t *testing.T) {
	sig := make([]uint32, 64)
	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Size: 1024, Signature: sig},
		},
	}

	selector := &LSHBandingSelector{
		Bands:       16,
		RowsPerBand: 4,
		MinBlobSize: 100,
		MaxBlobSize: 10000,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	if len(assignments.assignments) != 0 {
		t.Errorf("expected no assignments for single blob, got %d",
			len(assignments.assignments))
	}
}

func TestLSHBandingSelectorEmptyBlobSet(t *testing.T) {
	blobs := &testBlobSet{blobs: nil}

	selector := &LSHBandingSelector{
		Bands:       16,
		RowsPerBand: 4,
		MinBlobSize: 100,
		MaxBlobSize: 10000,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	if len(assignments.assignments) != 0 {
		t.Errorf("expected no assignments for empty set, got %d",
			len(assignments.assignments))
	}
}
