//go:build test && debug

package inventory_archive

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
)

func TestV1RoundTripFullEntriesOnly(t *testing.T) {
	var buf bytes.Buffer
	hashFormatId := "sha256"
	ct := compression_type.CompressionTypeNone

	writer, err := NewDataWriterV1(&buf, hashFormatId, ct, 0)
	if err != nil {
		t.Fatalf("NewDataWriterV1: %v", err)
	}

	entries := []struct {
		data []byte
		hash []byte
	}{
		{
			data: []byte("hello world"),
			hash: sha256Hash([]byte("hello world")),
		},
		{
			data: []byte("second entry with more data"),
			hash: sha256Hash([]byte("second entry with more data")),
		},
	}

	for _, e := range entries {
		if err := writer.WriteFullEntry(e.hash, e.data); err != nil {
			t.Fatalf("WriteFullEntry: %v", err)
		}
	}

	checksum, writtenEntries, err := writer.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	if len(checksum) != sha256.Size {
		t.Fatalf("expected checksum length %d, got %d", sha256.Size, len(checksum))
	}

	if len(writtenEntries) != len(entries) {
		t.Fatalf(
			"expected %d entries, got %d",
			len(entries),
			len(writtenEntries),
		)
	}

	for i, we := range writtenEntries {
		if !bytes.Equal(we.Hash, entries[i].hash) {
			t.Errorf("entry %d: hash mismatch", i)
		}

		if we.EntryType != EntryTypeFull {
			t.Errorf("entry %d: expected EntryTypeFull, got %d", i, we.EntryType)
		}

		if we.UncompressedSize != uint64(len(entries[i].data)) {
			t.Errorf(
				"entry %d: uncompressed size %d != %d",
				i,
				we.UncompressedSize,
				len(entries[i].data),
			)
		}
	}

	reader, err := NewDataReaderV1(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("NewDataReaderV1: %v", err)
	}

	if reader.HashFormatId() != hashFormatId {
		t.Fatalf(
			"hash format id: got %q, want %q",
			reader.HashFormatId(),
			hashFormatId,
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
		if !bytes.Equal(re.Hash, entries[i].hash) {
			t.Errorf("entry %d: hash mismatch on read", i)
		}

		if re.EntryType != EntryTypeFull {
			t.Errorf("entry %d: expected EntryTypeFull, got %d", i, re.EntryType)
		}

		if !bytes.Equal(re.Data, entries[i].data) {
			t.Errorf(
				"entry %d: data mismatch on read: got %q, want %q",
				i,
				re.Data,
				entries[i].data,
			)
		}

		if re.UncompressedSize != uint64(len(entries[i].data)) {
			t.Errorf(
				"entry %d: uncompressed size mismatch: %d != %d",
				i,
				re.UncompressedSize,
				len(entries[i].data),
			)
		}
	}

	// Test ReadEntryAt using offsets from the writer
	for i, we := range writtenEntries {
		re, err := reader.ReadEntryAt(we.Offset)
		if err != nil {
			t.Fatalf("ReadEntryAt(%d): %v", we.Offset, err)
		}

		if !bytes.Equal(re.Hash, entries[i].hash) {
			t.Errorf("ReadEntryAt entry %d: hash mismatch", i)
		}

		if !bytes.Equal(re.Data, entries[i].data) {
			t.Errorf(
				"ReadEntryAt entry %d: data mismatch: got %q, want %q",
				i,
				re.Data,
				entries[i].data,
			)
		}
	}

	// Validate checksum
	if err := reader.Validate(); err != nil {
		t.Fatalf("Validate should succeed: %v", err)
	}
}

func TestV1RoundTripWithDelta(t *testing.T) {
	var buf bytes.Buffer
	hashFormatId := "sha256"
	ct := compression_type.CompressionTypeNone

	writer, err := NewDataWriterV1(&buf, hashFormatId, ct, FlagHasDeltas)
	if err != nil {
		t.Fatalf("NewDataWriterV1: %v", err)
	}

	fullData := []byte("the base content for delta")
	fullHash := sha256Hash(fullData)

	deltaPayload := []byte("raw delta bytes here")
	deltaHash := sha256Hash([]byte("reconstructed content"))
	uncompressedSize := uint64(len("reconstructed content"))

	if err := writer.WriteFullEntry(fullHash, fullData); err != nil {
		t.Fatalf("WriteFullEntry: %v", err)
	}

	if err := writer.WriteDeltaEntry(
		deltaHash,
		DeltaAlgorithmByteBsdiff,
		fullHash,
		uncompressedSize,
		deltaPayload,
	); err != nil {
		t.Fatalf("WriteDeltaEntry: %v", err)
	}

	checksum, writtenEntries, err := writer.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	if len(checksum) != sha256.Size {
		t.Fatalf("expected checksum length %d, got %d", sha256.Size, len(checksum))
	}

	if len(writtenEntries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(writtenEntries))
	}

	if writtenEntries[0].EntryType != EntryTypeFull {
		t.Errorf("entry 0: expected EntryTypeFull, got %d", writtenEntries[0].EntryType)
	}

	if writtenEntries[1].EntryType != EntryTypeDelta {
		t.Errorf("entry 1: expected EntryTypeDelta, got %d", writtenEntries[1].EntryType)
	}

	if writtenEntries[1].DeltaAlgorithm != DeltaAlgorithmByteBsdiff {
		t.Errorf(
			"entry 1: expected delta algorithm %d, got %d",
			DeltaAlgorithmByteBsdiff,
			writtenEntries[1].DeltaAlgorithm,
		)
	}

	reader, err := NewDataReaderV1(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("NewDataReaderV1: %v", err)
	}

	readEntries, err := reader.ReadAllEntries()
	if err != nil {
		t.Fatalf("ReadAllEntries: %v", err)
	}

	if len(readEntries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(readEntries))
	}

	// Check full entry
	if readEntries[0].EntryType != EntryTypeFull {
		t.Errorf("read entry 0: expected EntryTypeFull, got %d", readEntries[0].EntryType)
	}

	if !bytes.Equal(readEntries[0].Data, fullData) {
		t.Errorf(
			"read entry 0: data mismatch: got %q, want %q",
			readEntries[0].Data,
			fullData,
		)
	}

	// Check delta entry
	if readEntries[1].EntryType != EntryTypeDelta {
		t.Errorf("read entry 1: expected EntryTypeDelta, got %d", readEntries[1].EntryType)
	}

	if readEntries[1].DeltaAlgorithm != DeltaAlgorithmByteBsdiff {
		t.Errorf(
			"read entry 1: expected delta algorithm %d, got %d",
			DeltaAlgorithmByteBsdiff,
			readEntries[1].DeltaAlgorithm,
		)
	}

	if !bytes.Equal(readEntries[1].BaseHash, fullHash) {
		t.Errorf("read entry 1: base hash mismatch")
	}

	if !bytes.Equal(readEntries[1].Data, deltaPayload) {
		t.Errorf(
			"read entry 1: delta payload mismatch: got %q, want %q",
			readEntries[1].Data,
			deltaPayload,
		)
	}

	if readEntries[1].UncompressedSize != uncompressedSize {
		t.Errorf(
			"read entry 1: uncompressed size %d != %d",
			readEntries[1].UncompressedSize,
			uncompressedSize,
		)
	}

	if err := reader.Validate(); err != nil {
		t.Fatalf("Validate should succeed: %v", err)
	}
}

func TestV1HeaderFlags(t *testing.T) {
	// Test 1: full entries only, flags = 0 => no FlagHasDeltas
	var buf1 bytes.Buffer
	hashFormatId := "sha256"
	ct := compression_type.CompressionTypeNone

	writer1, err := NewDataWriterV1(&buf1, hashFormatId, ct, 0)
	if err != nil {
		t.Fatalf("NewDataWriterV1: %v", err)
	}

	if err := writer1.WriteFullEntry(
		sha256Hash([]byte("test")),
		[]byte("test"),
	); err != nil {
		t.Fatalf("WriteFullEntry: %v", err)
	}

	if _, _, err := writer1.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	reader1, err := NewDataReaderV1(bytes.NewReader(buf1.Bytes()))
	if err != nil {
		t.Fatalf("NewDataReaderV1: %v", err)
	}

	if reader1.Flags()&FlagHasDeltas != 0 {
		t.Error("expected FlagHasDeltas to NOT be set for full-only archive")
	}

	// Test 2: flags = FlagHasDeltas => FlagHasDeltas should be set
	var buf2 bytes.Buffer

	writer2, err := NewDataWriterV1(&buf2, hashFormatId, ct, FlagHasDeltas)
	if err != nil {
		t.Fatalf("NewDataWriterV1: %v", err)
	}

	if err := writer2.WriteFullEntry(
		sha256Hash([]byte("base")),
		[]byte("base"),
	); err != nil {
		t.Fatalf("WriteFullEntry: %v", err)
	}

	if err := writer2.WriteDeltaEntry(
		sha256Hash([]byte("target")),
		DeltaAlgorithmByteBsdiff,
		sha256Hash([]byte("base")),
		uint64(len("target")),
		[]byte("delta payload"),
	); err != nil {
		t.Fatalf("WriteDeltaEntry: %v", err)
	}

	if _, _, err := writer2.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Read back the flags from the header
	data := buf2.Bytes()
	// Header layout: magic(4) + version(2) + hash_format_id_len(1) +
	//   hash_format_id(6 for "sha256") + default_encoding(1) + flags(2)
	flagsOffset := 4 + 2 + 1 + len("sha256") + 1
	flags := binary.BigEndian.Uint16(data[flagsOffset : flagsOffset+2])

	if flags&FlagHasDeltas == 0 {
		t.Error("expected FlagHasDeltas to be set for archive with deltas")
	}

	reader2, err := NewDataReaderV1(bytes.NewReader(buf2.Bytes()))
	if err != nil {
		t.Fatalf("NewDataReaderV1: %v", err)
	}

	if reader2.Flags()&FlagHasDeltas == 0 {
		t.Error("expected FlagHasDeltas from reader for archive with deltas")
	}
}
