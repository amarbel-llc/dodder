//go:build test && debug

package inventory_archive

import (
	"bytes"
	"math"
	"testing"
)

func TestGearCDCMinHashComputerSignatureLen(t *testing.T) {
	computer := &GearCDCMinHashComputer{
		AvgChunkSize: 64,
		MinChunkSize: 16,
		MaxChunkSize: 256,
		K:            64,
	}

	if computer.SignatureLen() != 64 {
		t.Errorf("expected 64, got %d", computer.SignatureLen())
	}
}

func TestGearCDCMinHashComputerSimilarBlobs(t *testing.T) {
	computer := &GearCDCMinHashComputer{
		AvgChunkSize: 64,
		MinChunkSize: 16,
		MaxChunkSize: 256,
		K:            128,
	}

	original := make([]byte, 2048)
	for i := range original {
		original[i] = byte(i * 7)
	}

	edited := make([]byte, len(original))
	copy(edited, original)
	for i := 1000; i < 1100; i++ {
		edited[i] = byte(i * 13)
	}

	sigA, err := computer.ComputeSignature(bytes.NewReader(original))
	if err != nil {
		t.Fatal(err)
	}

	sigB, err := computer.ComputeSignature(bytes.NewReader(edited))
	if err != nil {
		t.Fatal(err)
	}

	similarity := MinHashJaccard(sigA, sigB)

	if similarity < 0.5 {
		t.Errorf("similar blobs have low similarity: %.3f", similarity)
	}
}

func TestGearCDCMinHashComputerDissimilarBlobs(t *testing.T) {
	computer := &GearCDCMinHashComputer{
		AvgChunkSize: 64,
		MinChunkSize: 16,
		MaxChunkSize: 256,
		K:            128,
	}

	blobA := make([]byte, 1024)
	for i := range blobA {
		blobA[i] = byte(i * 3)
	}

	blobB := make([]byte, 1024)
	for i := range blobB {
		blobB[i] = byte(i*17 + 128)
	}

	sigA, err := computer.ComputeSignature(bytes.NewReader(blobA))
	if err != nil {
		t.Fatal(err)
	}

	sigB, err := computer.ComputeSignature(bytes.NewReader(blobB))
	if err != nil {
		t.Fatal(err)
	}

	similarity := MinHashJaccard(sigA, sigB)

	if similarity > 0.3 {
		t.Errorf("dissimilar blobs have high similarity: %.3f", similarity)
	}
}

func TestGearCDCMinHashComputerIdenticalBlobs(t *testing.T) {
	computer := &GearCDCMinHashComputer{
		AvgChunkSize: 64,
		MinChunkSize: 16,
		MaxChunkSize: 256,
		K:            64,
	}

	data := []byte("the quick brown fox jumps over the lazy dog, repeatedly")

	sigA, err := computer.ComputeSignature(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	sigB, err := computer.ComputeSignature(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	similarity := MinHashJaccard(sigA, sigB)

	if similarity != 1.0 {
		t.Errorf("identical blobs should have similarity 1.0, got %.3f", similarity)
	}
}

func TestGearCDCMinHashComputerEmptyBlob(t *testing.T) {
	computer := &GearCDCMinHashComputer{
		AvgChunkSize: 64,
		MinChunkSize: 16,
		MaxChunkSize: 256,
		K:            16,
	}

	sig, err := computer.ComputeSignature(bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}

	if len(sig) != 16 {
		t.Errorf("expected len 16, got %d", len(sig))
	}

	for i, v := range sig {
		if v != math.MaxUint32 {
			t.Errorf("position %d: expected MaxUint32 for empty blob, got %d", i, v)
		}
	}
}
