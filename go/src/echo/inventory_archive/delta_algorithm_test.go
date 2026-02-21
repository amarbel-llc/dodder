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
	b, err := DeltaAlgorithmByteForName("xdelta")
	if err != nil {
		t.Fatalf("expected xdelta byte, got error: %v", err)
	}

	if b != DeltaAlgorithmByteXdelta {
		t.Errorf("expected byte %d, got %d", DeltaAlgorithmByteXdelta, b)
	}
}

func TestDeltaAlgorithmNameUnknown(t *testing.T) {
	_, err := DeltaAlgorithmByteForName("unknown")
	if err == nil {
		t.Fatal("expected error for unknown algorithm name")
	}
}
