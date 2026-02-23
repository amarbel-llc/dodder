//go:build test && debug

package blob_stores

import (
	"testing"
)

func TestSplitBlobChunksUnlimited(t *testing.T) {
	metas := []packedBlobMeta{
		{digest: []byte{0x01}, size: 100},
		{digest: []byte{0x02}, size: 200},
		{digest: []byte{0x03}, size: 300},
	}

	chunks := splitBlobChunks(metas, 0)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for unlimited, got %d", len(chunks))
	}

	if len(chunks[0]) != 3 {
		t.Fatalf("expected 3 blobs in chunk, got %d", len(chunks[0]))
	}
}

func TestSplitBlobChunksSplitsAtLimit(t *testing.T) {
	metas := []packedBlobMeta{
		{digest: []byte{0x01}, size: 100},
		{digest: []byte{0x02}, size: 100},
		{digest: []byte{0x03}, size: 100},
		{digest: []byte{0x04}, size: 100},
	}

	// Limit 250 means first chunk gets blobs 1+2 (200 bytes),
	// blob 3 would push to 300 so it starts a new chunk.
	chunks := splitBlobChunks(metas, 250)

	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	if len(chunks[0]) != 2 {
		t.Fatalf("expected 2 blobs in first chunk, got %d", len(chunks[0]))
	}

	if len(chunks[1]) != 2 {
		t.Fatalf("expected 2 blobs in second chunk, got %d", len(chunks[1]))
	}
}

func TestSplitBlobChunksSingleBlobExceedsLimit(t *testing.T) {
	metas := []packedBlobMeta{
		{digest: []byte{0x01}, size: 500},
		{digest: []byte{0x02}, size: 100},
	}

	// Limit 200 but first blob is 500 -- it still gets its own chunk.
	chunks := splitBlobChunks(metas, 200)

	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	if len(chunks[0]) != 1 {
		t.Fatalf("expected 1 blob in first chunk (oversized), got %d", len(chunks[0]))
	}

	if len(chunks[1]) != 1 {
		t.Fatalf("expected 1 blob in second chunk, got %d", len(chunks[1]))
	}
}

func TestSplitBlobChunksEmpty(t *testing.T) {
	chunks := splitBlobChunks(nil, 100)

	if len(chunks) != 0 {
		t.Fatalf("expected 0 chunks for empty input, got %d", len(chunks))
	}
}
