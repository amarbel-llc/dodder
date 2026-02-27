//go:build test && debug

package inventory_archive

import (
	"bytes"
	"testing"
)

func TestGearCDCProducesChunks(t *testing.T) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i * 37)
	}

	chunks := GearCDCChunks(data, 16, 256, 64)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	var reconstructed []byte
	for _, c := range chunks {
		reconstructed = append(reconstructed, c...)
	}

	if !bytes.Equal(data, reconstructed) {
		t.Fatal("chunks do not reconstruct original data")
	}
}

func TestGearCDCRespectsMinChunkSize(t *testing.T) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i)
	}

	minSize := 32
	chunks := GearCDCChunks(data, minSize, 256, 64)

	for i, c := range chunks {
		if i < len(chunks)-1 && len(c) < minSize {
			t.Errorf("chunk %d has size %d < min %d", i, len(c), minSize)
		}
	}
}

func TestGearCDCRespectsMaxChunkSize(t *testing.T) {
	data := make([]byte, 4096)
	for i := range data {
		data[i] = byte(i)
	}

	maxSize := 128
	chunks := GearCDCChunks(data, 16, maxSize, 64)

	for i, c := range chunks {
		if len(c) > maxSize {
			t.Errorf("chunk %d has size %d > max %d", i, len(c), maxSize)
		}
	}
}

func TestGearCDCInsertionOnlyAffectsNearbyChunks(t *testing.T) {
	original := make([]byte, 2048)
	for i := range original {
		original[i] = byte(i * 7)
	}

	insertion := make([]byte, len(original)+10)
	copy(insertion[:1024], original[:1024])
	copy(insertion[1024:1034], []byte("INSERTDATA"))
	copy(insertion[1034:], original[1024:])

	origChunks := GearCDCChunks(original, 16, 256, 64)
	insChunks := GearCDCChunks(insertion, 16, 256, 64)

	origSet := make(map[string]bool)
	for _, c := range origChunks {
		origSet[string(c)] = true
	}

	matching := 0
	for _, c := range insChunks {
		if origSet[string(c)] {
			matching++
		}
	}

	matchRatio := float64(matching) / float64(len(origChunks))
	if matchRatio < 0.5 {
		t.Errorf(
			"insertion disrupted too many chunks: %d/%d matching (%.1f%%)",
			matching, len(origChunks), matchRatio*100,
		)
	}
}

func TestGearCDCEmptyInput(t *testing.T) {
	chunks := GearCDCChunks(nil, 16, 256, 64)
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty input, got %d", len(chunks))
	}
}

func TestGearCDCSmallInput(t *testing.T) {
	data := []byte("hello")
	chunks := GearCDCChunks(data, 16, 256, 64)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for small input, got %d", len(chunks))
	}

	if !bytes.Equal(chunks[0], data) {
		t.Fatal("single chunk should equal input")
	}
}
