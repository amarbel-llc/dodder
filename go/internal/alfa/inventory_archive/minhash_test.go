//go:build test && debug

package inventory_archive

import (
	"math"
	"testing"
)

func TestMinHashIdenticalSetsProduceIdenticalSignatures(t *testing.T) {
	set := []uint32{100, 200, 300, 400, 500}

	sig1 := MinHashSignature(set, 16)
	sig2 := MinHashSignature(set, 16)

	for i := range sig1 {
		if sig1[i] != sig2[i] {
			t.Fatalf("position %d: %d != %d", i, sig1[i], sig2[i])
		}
	}
}

func TestMinHashDisjointSetsProduceDifferentSignatures(t *testing.T) {
	setA := []uint32{1, 2, 3, 4, 5}
	setB := []uint32{1001, 1002, 1003, 1004, 1005}

	sigA := MinHashSignature(setA, 64)
	sigB := MinHashSignature(setB, 64)

	matches := 0
	for i := range sigA {
		if sigA[i] == sigB[i] {
			matches++
		}
	}

	if float64(matches)/float64(len(sigA)) > 0.2 {
		t.Errorf("disjoint sets have too many matches: %d/%d", matches, len(sigA))
	}
}

func TestMinHashEstimatesJaccardSimilarity(t *testing.T) {
	shared := make([]uint32, 80)
	for i := range shared {
		shared[i] = uint32(i)
	}

	setA := make([]uint32, 100)
	copy(setA[:80], shared)
	for i := 80; i < 100; i++ {
		setA[i] = uint32(1000 + i)
	}

	setB := make([]uint32, 100)
	copy(setB[:80], shared)
	for i := 80; i < 100; i++ {
		setB[i] = uint32(2000 + i)
	}

	expectedJaccard := 80.0 / 120.0

	k := 256
	sigA := MinHashSignature(setA, k)
	sigB := MinHashSignature(setB, k)

	estimated := MinHashJaccard(sigA, sigB)

	tolerance := 3 * math.Sqrt(expectedJaccard*(1-expectedJaccard)/float64(k))

	if math.Abs(estimated-expectedJaccard) > tolerance {
		t.Errorf(
			"estimated Jaccard %.3f too far from expected %.3f (tolerance %.3f)",
			estimated, expectedJaccard, tolerance,
		)
	}
}

func TestMinHashSignatureLenMatchesK(t *testing.T) {
	set := []uint32{1, 2, 3}
	sig := MinHashSignature(set, 32)

	if len(sig) != 32 {
		t.Errorf("expected signature length 32, got %d", len(sig))
	}
}

func TestMinHashEmptySet(t *testing.T) {
	sig := MinHashSignature(nil, 16)

	if len(sig) != 16 {
		t.Errorf("expected signature length 16, got %d", len(sig))
	}

	for i, v := range sig {
		if v != math.MaxUint32 {
			t.Errorf("position %d: expected MaxUint32, got %d", i, v)
		}
	}
}
