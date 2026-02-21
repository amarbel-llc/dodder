//go:build test && debug

package inventory_archive

import (
	"testing"
)

func TestDeltaAlgorithmRegistryUnknown(t *testing.T) {
	_, err := DeltaAlgorithmForByte(0xFF)
	if err == nil {
		t.Fatal("expected error for unknown algorithm byte")
	}
}

func TestDeltaAlgorithmNameLookup(t *testing.T) {
	b, err := DeltaAlgorithmByteForName("bsdiff")
	if err != nil {
		t.Fatalf("expected bsdiff byte, got error: %v", err)
	}

	if b != DeltaAlgorithmByteBsdiff {
		t.Errorf("expected byte %d, got %d", DeltaAlgorithmByteBsdiff, b)
	}
}

func TestDeltaAlgorithmNameUnknown(t *testing.T) {
	_, err := DeltaAlgorithmByteForName("unknown")
	if err == nil {
		t.Fatal("expected error for unknown algorithm name")
	}
}
