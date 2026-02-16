//go:build test && debug

package inventory_archive

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
)

func TestRoundTripNoCompression(t *testing.T) {
	var buf bytes.Buffer
	hashFormatId := "sha256"
	ct := compression_type.CompressionTypeNone

	writer, err := NewDataWriter(&buf, hashFormatId, ct)
	if err != nil {
		t.Fatalf("NewDataWriter: %v", err)
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
		{
			data: []byte("third"),
			hash: sha256Hash([]byte("third")),
		},
	}

	for _, e := range entries {
		if err := writer.WriteEntry(e.hash, e.data); err != nil {
			t.Fatalf("WriteEntry: %v", err)
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

		if we.UncompressedSize != uint64(len(entries[i].data)) {
			t.Errorf(
				"entry %d: uncompressed size %d != %d",
				i,
				we.UncompressedSize,
				len(entries[i].data),
			)
		}
	}

	reader, err := NewDataReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("NewDataReader: %v", err)
	}

	if reader.HashFormatId() != hashFormatId {
		t.Fatalf(
			"hash format id: got %q, want %q",
			reader.HashFormatId(),
			hashFormatId,
		)
	}

	if reader.CompressionType() != ct {
		t.Fatalf(
			"compression type: got %q, want %q",
			reader.CompressionType(),
			ct,
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
}

func TestRoundTripZstd(t *testing.T) {
	var buf bytes.Buffer
	hashFormatId := "sha256"
	ct := compression_type.CompressionTypeZstd

	writer, err := NewDataWriter(&buf, hashFormatId, ct)
	if err != nil {
		t.Fatalf("NewDataWriter: %v", err)
	}

	testData := [][]byte{
		[]byte("compressible data that repeats repeats repeats repeats"),
		[]byte("another block of data for testing compression"),
	}

	testHashes := make([][]byte, len(testData))
	for i, data := range testData {
		testHashes[i] = sha256Hash(data)
	}

	for i, data := range testData {
		if err := writer.WriteEntry(testHashes[i], data); err != nil {
			t.Fatalf("WriteEntry: %v", err)
		}
	}

	checksum, writtenEntries, err := writer.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	if len(checksum) == 0 {
		t.Fatal("expected non-empty checksum")
	}

	if len(writtenEntries) != len(testData) {
		t.Fatalf(
			"expected %d entries, got %d",
			len(testData),
			len(writtenEntries),
		)
	}

	reader, err := NewDataReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("NewDataReader: %v", err)
	}

	if reader.CompressionType() != ct {
		t.Fatalf(
			"compression type: got %q, want %q",
			reader.CompressionType(),
			ct,
		)
	}

	readEntries, err := reader.ReadAllEntries()
	if err != nil {
		t.Fatalf("ReadAllEntries: %v", err)
	}

	if len(readEntries) != len(testData) {
		t.Fatalf(
			"expected %d entries, got %d",
			len(testData),
			len(readEntries),
		)
	}

	for i, re := range readEntries {
		if !bytes.Equal(re.Hash, testHashes[i]) {
			t.Errorf("entry %d: hash mismatch on read", i)
		}

		if !bytes.Equal(re.Data, testData[i]) {
			t.Errorf(
				"entry %d: data mismatch on read: got %q, want %q",
				i,
				re.Data,
				testData[i],
			)
		}

		if re.UncompressedSize != uint64(len(testData[i])) {
			t.Errorf(
				"entry %d: uncompressed size mismatch: %d != %d",
				i,
				re.UncompressedSize,
				len(testData[i]),
			)
		}
	}
}

func sha256Hash(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}
