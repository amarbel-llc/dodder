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

func makeTestCacheEntries(count int) []CacheEntry {
	entries := make([]CacheEntry, count)

	for i := range count {
		data := []byte(fmt.Sprintf("test-cache-entry-%04d", i))
		h := sha256.Sum256(data)

		archiveData := []byte(fmt.Sprintf("archive-%04d", i))
		archiveChecksum := sha256.Sum256(archiveData)

		entries[i] = CacheEntry{
			Hash:            h[:],
			ArchiveChecksum: archiveChecksum[:],
			Offset:          uint64(i * 2000),
			CompressedSize:  uint64(200 + i),
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return bytes.Compare(entries[i].Hash, entries[j].Hash) < 0
	})

	return entries
}

func TestCacheRoundTrip(t *testing.T) {
	hashFormatId := "sha256"
	entries := makeTestCacheEntries(20)

	var buf bytes.Buffer

	checksum, err := WriteCache(&buf, hashFormatId, entries)
	if err != nil {
		t.Fatalf("WriteCache: %v", err)
	}

	if len(checksum) != sha256.Size {
		t.Fatalf(
			"expected checksum length %d, got %d",
			sha256.Size,
			len(checksum),
		)
	}

	data := buf.Bytes()
	reader, err := NewCacheReader(
		bytes.NewReader(data),
		int64(len(data)),
		hashFormatId,
	)
	if err != nil {
		t.Fatalf("NewCacheReader: %v", err)
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
	}
}

func TestCacheToMap(t *testing.T) {
	hashFormatId := "sha256"
	entries := makeTestCacheEntries(10)

	var buf bytes.Buffer

	if _, err := WriteCache(&buf, hashFormatId, entries); err != nil {
		t.Fatalf("WriteCache: %v", err)
	}

	data := buf.Bytes()
	reader, err := NewCacheReader(
		bytes.NewReader(data),
		int64(len(data)),
		hashFormatId,
	)
	if err != nil {
		t.Fatalf("NewCacheReader: %v", err)
	}

	readEntries, err := reader.ReadAllEntries()
	if err != nil {
		t.Fatalf("ReadAllEntries: %v", err)
	}

	m := ToMap(readEntries)

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
	}
}

func TestCacheEmpty(t *testing.T) {
	hashFormatId := "sha256"
	var entries []CacheEntry

	var buf bytes.Buffer

	checksum, err := WriteCache(&buf, hashFormatId, entries)
	if err != nil {
		t.Fatalf("WriteCache: %v", err)
	}

	if len(checksum) != sha256.Size {
		t.Fatalf(
			"expected checksum length %d, got %d",
			sha256.Size,
			len(checksum),
		)
	}

	data := buf.Bytes()
	reader, err := NewCacheReader(
		bytes.NewReader(data),
		int64(len(data)),
		hashFormatId,
	)
	if err != nil {
		t.Fatalf("NewCacheReader: %v", err)
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

	m := ToMap(readEntries)

	if len(m) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(m))
	}
}

func TestCacheValidateChecksum(t *testing.T) {
	hashFormatId := "sha256"
	entries := makeTestCacheEntries(5)

	var buf bytes.Buffer

	if _, err := WriteCache(&buf, hashFormatId, entries); err != nil {
		t.Fatalf("WriteCache: %v", err)
	}

	data := buf.Bytes()

	// Valid data should pass
	reader, err := NewCacheReader(
		bytes.NewReader(data),
		int64(len(data)),
		hashFormatId,
	)
	if err != nil {
		t.Fatalf("NewCacheReader: %v", err)
	}

	if err := reader.Validate(); err != nil {
		t.Fatalf("Validate should succeed on valid cache: %v", err)
	}

	// Corrupt a byte
	corrupted := make([]byte, len(data))
	copy(corrupted, data)
	midpoint := len(corrupted) / 2
	corrupted[midpoint] ^= 0xFF

	reader, err = NewCacheReader(
		bytes.NewReader(corrupted),
		int64(len(corrupted)),
		hashFormatId,
	)
	if err != nil {
		t.Fatalf("NewCacheReader on corrupted data: %v", err)
	}

	if err := reader.Validate(); err == nil {
		t.Fatal("Validate should fail on corrupted cache")
	}
}

func TestCacheWriterRejectsUnsortedEntries(t *testing.T) {
	hashFormatId := "sha256"
	entries := makeTestCacheEntries(5)

	// Reverse the sorted entries to make them unsorted
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	var buf bytes.Buffer

	if _, err := WriteCache(&buf, hashFormatId, entries); err == nil {
		t.Fatal("WriteCache should reject unsorted entries")
	}
}
