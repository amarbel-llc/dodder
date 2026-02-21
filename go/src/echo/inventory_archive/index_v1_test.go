//go:build test && debug

package inventory_archive

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"sort"
	"testing"
)

func makeTestIndexV1Entries(count int) []IndexEntryV1 {
	entries := make([]IndexEntryV1, count)

	for i := range count {
		data := []byte(fmt.Sprintf("test-data-entry-v1-%04d", i))
		h := sha256.Sum256(data)
		entries[i] = IndexEntryV1{
			Hash:           h[:],
			PackOffset:     uint64(i * 1000),
			CompressedSize: uint64(100 + i),
			EntryType:      EntryTypeFull,
			BaseOffset:     0,
		}

		// Make some entries delta
		if i%3 == 0 && i > 0 {
			entries[i].EntryType = EntryTypeDelta
			entries[i].BaseOffset = uint64((i - 1) * 1000)
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return bytes.Compare(entries[i].Hash, entries[j].Hash) < 0
	})

	return entries
}

func TestIndexV1RoundTrip(t *testing.T) {
	hashFormatId := "sha256"
	entries := makeTestIndexV1Entries(20)

	var buf bytes.Buffer

	checksum, err := WriteIndexV1(&buf, hashFormatId, entries)
	if err != nil {
		t.Fatalf("WriteIndexV1: %v", err)
	}

	if len(checksum) != sha256.Size {
		t.Fatalf(
			"expected checksum length %d, got %d",
			sha256.Size,
			len(checksum),
		)
	}

	data := buf.Bytes()
	reader, err := NewIndexReaderV1(
		bytes.NewReader(data),
		int64(len(data)),
		hashFormatId,
	)
	if err != nil {
		t.Fatalf("NewIndexReaderV1: %v", err)
	}

	if reader.HashFormatId() != hashFormatId {
		t.Fatalf(
			"hash format id: got %q, want %q",
			reader.HashFormatId(),
			hashFormatId,
		)
	}

	if reader.EntryCount() != uint64(len(entries)) {
		t.Fatalf(
			"entry count: got %d, want %d",
			reader.EntryCount(),
			len(entries),
		)
	}

	readEntries, err := reader.ReadAllEntries()
	if err != nil {
		t.Fatalf("ReadAllEntries: %v", err)
	}

	if len(readEntries) != len(entries) {
		t.Fatalf(
			"expected %d entries, got %d",
			len(entries),
			len(readEntries),
		)
	}

	for i, re := range readEntries {
		if !bytes.Equal(re.Hash, entries[i].Hash) {
			t.Errorf("entry %d: hash mismatch", i)
		}

		if re.PackOffset != entries[i].PackOffset {
			t.Errorf(
				"entry %d: pack offset %d != %d",
				i,
				re.PackOffset,
				entries[i].PackOffset,
			)
		}

		if re.CompressedSize != entries[i].CompressedSize {
			t.Errorf(
				"entry %d: compressed size %d != %d",
				i,
				re.CompressedSize,
				entries[i].CompressedSize,
			)
		}

		if re.EntryType != entries[i].EntryType {
			t.Errorf(
				"entry %d: entry type %d != %d",
				i,
				re.EntryType,
				entries[i].EntryType,
			)
		}

		if re.BaseOffset != entries[i].BaseOffset {
			t.Errorf(
				"entry %d: base offset %d != %d",
				i,
				re.BaseOffset,
				entries[i].BaseOffset,
			)
		}
	}
}

func TestIndexV1Lookup(t *testing.T) {
	hashFormatId := "sha256"
	entries := makeTestIndexV1Entries(10)

	var buf bytes.Buffer

	if _, err := WriteIndexV1(&buf, hashFormatId, entries); err != nil {
		t.Fatalf("WriteIndexV1: %v", err)
	}

	data := buf.Bytes()
	reader, err := NewIndexReaderV1(
		bytes.NewReader(data),
		int64(len(data)),
		hashFormatId,
	)
	if err != nil {
		t.Fatalf("NewIndexReaderV1: %v", err)
	}

	// Look up each entry
	for i, entry := range entries {
		packOffset, compressedSize, entryType, baseOffset, found, lookupErr := reader.LookupHash(
			entry.Hash,
		)
		if lookupErr != nil {
			t.Fatalf("LookupHash entry %d: %v", i, lookupErr)
		}

		if !found {
			t.Errorf("entry %d: not found", i)
			continue
		}

		if packOffset != entry.PackOffset {
			t.Errorf(
				"entry %d: pack offset %d != %d",
				i,
				packOffset,
				entry.PackOffset,
			)
		}

		if compressedSize != entry.CompressedSize {
			t.Errorf(
				"entry %d: compressed size %d != %d",
				i,
				compressedSize,
				entry.CompressedSize,
			)
		}

		if entryType != entry.EntryType {
			t.Errorf(
				"entry %d: entry type %d != %d",
				i,
				entryType,
				entry.EntryType,
			)
		}

		if baseOffset != entry.BaseOffset {
			t.Errorf(
				"entry %d: base offset %d != %d",
				i,
				baseOffset,
				entry.BaseOffset,
			)
		}
	}

	// Look up a hash that does not exist
	nonExistentHash := sha256.Sum256([]byte("this-hash-does-not-exist"))
	_, _, _, _, found, err := reader.LookupHash(nonExistentHash[:])
	if err != nil {
		t.Fatalf("LookupHash non-existent: %v", err)
	}

	if found {
		t.Error("expected non-existent hash to not be found")
	}
}

func TestIndexV1FanOut(t *testing.T) {
	hashFormatId := "sha256"
	entries := makeTestIndexV1Entries(15)

	var buf bytes.Buffer

	if _, err := WriteIndexV1(&buf, hashFormatId, entries); err != nil {
		t.Fatalf("WriteIndexV1: %v", err)
	}

	data := buf.Bytes()
	reader, err := NewIndexReaderV1(
		bytes.NewReader(data),
		int64(len(data)),
		hashFormatId,
	)
	if err != nil {
		t.Fatalf("NewIndexReaderV1: %v", err)
	}

	// Build expected fan-out from entries
	var expectedFanOut [256]uint64
	for _, entry := range entries {
		firstByte := entry.Hash[0]
		for j := int(firstByte); j < 256; j++ {
			expectedFanOut[j]++
		}
	}

	fanOut := reader.FanOut()
	for i := range 256 {
		if fanOut[i] != expectedFanOut[i] {
			t.Errorf(
				"fan-out[%d]: got %d, want %d",
				i,
				fanOut[i],
				expectedFanOut[i],
			)
		}
	}

	// Verify last fan-out entry equals total entry count
	if fanOut[255] != uint64(len(entries)) {
		t.Errorf(
			"fan-out[255]: got %d, want %d",
			fanOut[255],
			len(entries),
		)
	}
}

func TestIndexV1Empty(t *testing.T) {
	hashFormatId := "sha256"
	var entries []IndexEntryV1

	var buf bytes.Buffer

	checksum, err := WriteIndexV1(&buf, hashFormatId, entries)
	if err != nil {
		t.Fatalf("WriteIndexV1: %v", err)
	}

	if len(checksum) != sha256.Size {
		t.Fatalf(
			"expected checksum length %d, got %d",
			sha256.Size,
			len(checksum),
		)
	}

	data := buf.Bytes()
	reader, err := NewIndexReaderV1(
		bytes.NewReader(data),
		int64(len(data)),
		hashFormatId,
	)
	if err != nil {
		t.Fatalf("NewIndexReaderV1: %v", err)
	}

	if reader.EntryCount() != 0 {
		t.Fatalf("expected 0 entries, got %d", reader.EntryCount())
	}

	readEntries, err := reader.ReadAllEntries()
	if err != nil {
		t.Fatalf("ReadAllEntries: %v", err)
	}

	if len(readEntries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(readEntries))
	}

	// Look up anything in empty index
	nonExistentHash := sha256.Sum256([]byte("anything"))
	_, _, _, _, found, lookupErr := reader.LookupHash(nonExistentHash[:])
	if lookupErr != nil {
		t.Fatalf("LookupHash in empty index: %v", lookupErr)
	}

	if found {
		t.Error("expected hash not found in empty index")
	}
}

func TestIndexV1ValidateChecksum(t *testing.T) {
	hashFormatId := "sha256"
	entries := makeTestIndexV1Entries(5)

	var buf bytes.Buffer

	if _, err := WriteIndexV1(&buf, hashFormatId, entries); err != nil {
		t.Fatalf("WriteIndexV1: %v", err)
	}

	data := buf.Bytes()

	// Valid data should pass
	reader, err := NewIndexReaderV1(
		bytes.NewReader(data),
		int64(len(data)),
		hashFormatId,
	)
	if err != nil {
		t.Fatalf("NewIndexReaderV1: %v", err)
	}

	if err := reader.Validate(); err != nil {
		t.Fatalf("Validate should succeed on valid index: %v", err)
	}

	// Corrupt a byte
	corrupted := make([]byte, len(data))
	copy(corrupted, data)
	midpoint := len(corrupted) / 2
	corrupted[midpoint] ^= 0xFF

	reader, err = NewIndexReaderV1(
		bytes.NewReader(corrupted),
		int64(len(corrupted)),
		hashFormatId,
	)
	if err != nil {
		t.Fatalf("NewIndexReaderV1 on corrupted data: %v", err)
	}

	if err := reader.Validate(); err == nil {
		t.Fatal("Validate should fail on corrupted index")
	}
}

func TestIndexV1WriterRejectsUnsortedEntries(t *testing.T) {
	hashFormatId := "sha256"
	entries := makeTestIndexV1Entries(5)

	// Reverse the sorted entries to make them unsorted
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	var buf bytes.Buffer

	if _, err := WriteIndexV1(&buf, hashFormatId, entries); err == nil {
		t.Fatal("WriteIndexV1 should reject unsorted entries")
	}
}
