//go:build test && debug

package inventory_archive

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"testing"
)

func makeTestCacheV1Entries(count int) []CacheEntryV1 {
	entries := make([]CacheEntryV1, count)

	for i := range count {
		data := []byte(fmt.Sprintf("test-cache-v1-entry-%04d", i))
		h := sha256.Sum256(data)

		archiveData := []byte(fmt.Sprintf("archive-v1-%04d", i))
		archiveChecksum := sha256.Sum256(archiveData)

		entries[i] = CacheEntryV1{
			Hash:            h[:],
			ArchiveChecksum: archiveChecksum[:],
			Offset:          uint64(i * 2000),
			CompressedSize:  uint64(200 + i),
			EntryType:       EntryTypeFull,
			BaseOffset:      0,
		}

		// Make some entries delta
		if i%3 == 0 && i > 0 {
			entries[i].EntryType = EntryTypeDelta
			entries[i].BaseOffset = uint64((i - 1) * 2000)
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return bytes.Compare(entries[i].Hash, entries[j].Hash) < 0
	})

	return entries
}

func TestCacheV1RoundTrip(t *testing.T) {
	hashFormatId := "sha256"
	entries := makeTestCacheV1Entries(20)

	var buf bytes.Buffer

	checksum, err := WriteCacheV1(&buf, hashFormatId, entries)
	if err != nil {
		t.Fatalf("WriteCacheV1: %v", err)
	}

	if len(checksum) != sha256.Size {
		t.Fatalf(
			"expected checksum length %d, got %d",
			sha256.Size,
			len(checksum),
		)
	}

	data := buf.Bytes()
	reader, err := NewCacheReaderV1(
		bytes.NewReader(data),
		int64(len(data)),
		hashFormatId,
	)
	if err != nil {
		t.Fatalf("NewCacheReaderV1: %v", err)
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

		if !bytes.Equal(re.ArchiveChecksum, entries[i].ArchiveChecksum) {
			t.Errorf("entry %d: archive checksum mismatch", i)
		}

		if re.Offset != entries[i].Offset {
			t.Errorf(
				"entry %d: offset %d != %d",
				i,
				re.Offset,
				entries[i].Offset,
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

func TestCacheV1ToMap(t *testing.T) {
	hashFormatId := "sha256"
	entries := makeTestCacheV1Entries(10)

	var buf bytes.Buffer

	if _, err := WriteCacheV1(&buf, hashFormatId, entries); err != nil {
		t.Fatalf("WriteCacheV1: %v", err)
	}

	data := buf.Bytes()
	reader, err := NewCacheReaderV1(
		bytes.NewReader(data),
		int64(len(data)),
		hashFormatId,
	)
	if err != nil {
		t.Fatalf("NewCacheReaderV1: %v", err)
	}

	readEntries, err := reader.ReadAllEntries()
	if err != nil {
		t.Fatalf("ReadAllEntries: %v", err)
	}

	m := ToMapV1(readEntries)

	if len(m) != len(entries) {
		t.Fatalf("expected map size %d, got %d", len(entries), len(m))
	}

	for _, entry := range entries {
		hexHash := hex.EncodeToString(entry.Hash)
		mapEntry, ok := m[hexHash]

		if !ok {
			t.Errorf("hash %s not found in map", hexHash)
			continue
		}

		if !bytes.Equal(mapEntry.ArchiveChecksum, entry.ArchiveChecksum) {
			t.Errorf(
				"hash %s: archive checksum mismatch",
				hexHash,
			)
		}

		if mapEntry.Offset != entry.Offset {
			t.Errorf(
				"hash %s: offset %d != %d",
				hexHash,
				mapEntry.Offset,
				entry.Offset,
			)
		}

		if mapEntry.CompressedSize != entry.CompressedSize {
			t.Errorf(
				"hash %s: compressed size %d != %d",
				hexHash,
				mapEntry.CompressedSize,
				entry.CompressedSize,
			)
		}

		if mapEntry.EntryType != entry.EntryType {
			t.Errorf(
				"hash %s: entry type %d != %d",
				hexHash,
				mapEntry.EntryType,
				entry.EntryType,
			)
		}

		if mapEntry.BaseOffset != entry.BaseOffset {
			t.Errorf(
				"hash %s: base offset %d != %d",
				hexHash,
				mapEntry.BaseOffset,
				entry.BaseOffset,
			)
		}
	}
}

func TestCacheV1Empty(t *testing.T) {
	hashFormatId := "sha256"
	var entries []CacheEntryV1

	var buf bytes.Buffer

	checksum, err := WriteCacheV1(&buf, hashFormatId, entries)
	if err != nil {
		t.Fatalf("WriteCacheV1: %v", err)
	}

	if len(checksum) != sha256.Size {
		t.Fatalf(
			"expected checksum length %d, got %d",
			sha256.Size,
			len(checksum),
		)
	}

	data := buf.Bytes()
	reader, err := NewCacheReaderV1(
		bytes.NewReader(data),
		int64(len(data)),
		hashFormatId,
	)
	if err != nil {
		t.Fatalf("NewCacheReaderV1: %v", err)
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

	m := ToMapV1(readEntries)

	if len(m) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(m))
	}
}

func TestCacheV1ValidateChecksum(t *testing.T) {
	hashFormatId := "sha256"
	entries := makeTestCacheV1Entries(5)

	var buf bytes.Buffer

	if _, err := WriteCacheV1(&buf, hashFormatId, entries); err != nil {
		t.Fatalf("WriteCacheV1: %v", err)
	}

	data := buf.Bytes()

	// Valid data should pass
	reader, err := NewCacheReaderV1(
		bytes.NewReader(data),
		int64(len(data)),
		hashFormatId,
	)
	if err != nil {
		t.Fatalf("NewCacheReaderV1: %v", err)
	}

	if err := reader.Validate(); err != nil {
		t.Fatalf("Validate should succeed on valid cache: %v", err)
	}

	// Corrupt a byte
	corrupted := make([]byte, len(data))
	copy(corrupted, data)
	midpoint := len(corrupted) / 2
	corrupted[midpoint] ^= 0xFF

	reader, err = NewCacheReaderV1(
		bytes.NewReader(corrupted),
		int64(len(corrupted)),
		hashFormatId,
	)
	if err != nil {
		t.Fatalf("NewCacheReaderV1 on corrupted data: %v", err)
	}

	if err := reader.Validate(); err == nil {
		t.Fatal("Validate should fail on corrupted cache")
	}
}

func TestCacheV1WriterRejectsUnsortedEntries(t *testing.T) {
	hashFormatId := "sha256"
	entries := makeTestCacheV1Entries(5)

	// Reverse the sorted entries to make them unsorted
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	var buf bytes.Buffer

	if _, err := WriteCacheV1(&buf, hashFormatId, entries); err == nil {
		t.Fatal("WriteCacheV1 should reject unsorted entries")
	}
}
