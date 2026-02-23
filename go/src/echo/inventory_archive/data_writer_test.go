//go:build test && debug

package inventory_archive

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
	"code.linenisgreat.com/dodder/go/src/delta/age"
)

func TestRoundTripNoCompression(t *testing.T) {
	var buf bytes.Buffer
	hashFormatId := "sha256"
	ct := compression_type.CompressionTypeNone

	writer, err := NewDataWriter(&buf, hashFormatId, ct, nil)
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

		if we.LogicalSize != uint64(len(entries[i].data)) {
			t.Errorf(
				"entry %d: uncompressed size %d != %d",
				i,
				we.LogicalSize,
				len(entries[i].data),
			)
		}
	}

	reader, err := NewDataReader(bytes.NewReader(buf.Bytes()), nil)
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

		if re.LogicalSize != uint64(len(entries[i].data)) {
			t.Errorf(
				"entry %d: uncompressed size mismatch: %d != %d",
				i,
				re.LogicalSize,
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

	writer, err := NewDataWriter(&buf, hashFormatId, ct, nil)
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

	reader, err := NewDataReader(bytes.NewReader(buf.Bytes()), nil)
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

		if re.LogicalSize != uint64(len(testData[i])) {
			t.Errorf(
				"entry %d: uncompressed size mismatch: %d != %d",
				i,
				re.LogicalSize,
				len(testData[i]),
			)
		}
	}
}

func TestValidateSucceeds(t *testing.T) {
	var buf bytes.Buffer
	hashFormatId := "sha256"
	ct := compression_type.CompressionTypeNone

	writer, err := NewDataWriter(&buf, hashFormatId, ct, nil)
	if err != nil {
		t.Fatalf("NewDataWriter: %v", err)
	}

	if err := writer.WriteEntry(
		sha256Hash([]byte("test")),
		[]byte("test"),
	); err != nil {
		t.Fatalf("WriteEntry: %v", err)
	}

	if _, _, err := writer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	reader, err := NewDataReader(bytes.NewReader(buf.Bytes()), nil)
	if err != nil {
		t.Fatalf("NewDataReader: %v", err)
	}

	if err := reader.Validate(); err != nil {
		t.Fatalf("Validate should succeed on valid archive: %v", err)
	}
}

func TestValidateDetectsCorruption(t *testing.T) {
	var buf bytes.Buffer
	hashFormatId := "sha256"
	ct := compression_type.CompressionTypeNone

	writer, err := NewDataWriter(&buf, hashFormatId, ct, nil)
	if err != nil {
		t.Fatalf("NewDataWriter: %v", err)
	}

	if err := writer.WriteEntry(
		sha256Hash([]byte("test")),
		[]byte("test"),
	); err != nil {
		t.Fatalf("WriteEntry: %v", err)
	}

	if _, _, err := writer.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Corrupt a byte in the data section (after the header)
	data := buf.Bytes()
	corrupted := make([]byte, len(data))
	copy(corrupted, data)
	// Flip a byte in the middle of the data (well past the header)
	midpoint := len(corrupted) / 2
	corrupted[midpoint] ^= 0xFF

	reader, err := NewDataReader(bytes.NewReader(corrupted), nil)
	if err != nil {
		t.Fatalf("NewDataReader: %v", err)
	}

	if err := reader.Validate(); err == nil {
		t.Fatal("Validate should fail on corrupted archive")
	}
}

func TestEmptyArchiveRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	hashFormatId := "sha256"
	ct := compression_type.CompressionTypeNone

	writer, err := NewDataWriter(&buf, hashFormatId, ct, nil)
	if err != nil {
		t.Fatalf("NewDataWriter: %v", err)
	}

	checksum, writtenEntries, err := writer.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	if len(checksum) != sha256.Size {
		t.Fatalf(
			"expected checksum length %d, got %d",
			sha256.Size,
			len(checksum),
		)
	}

	if len(writtenEntries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(writtenEntries))
	}

	reader, err := NewDataReader(bytes.NewReader(buf.Bytes()), nil)
	if err != nil {
		t.Fatalf("NewDataReader: %v", err)
	}

	readEntries, err := reader.ReadAllEntries()
	if err != nil {
		t.Fatalf("ReadAllEntries: %v", err)
	}

	if len(readEntries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(readEntries))
	}

	if err := reader.Validate(); err != nil {
		t.Fatalf("Validate should succeed on empty archive: %v", err)
	}
}

func TestEncryptedRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	hashFormatId := "sha256"
	ct := compression_type.CompressionTypeZstd

	// Use age X25519 for encryption
	var ageIdentity age.Identity
	if err := ageIdentity.GenerateIfNecessary(); err != nil {
		t.Fatal(err)
	}

	var encryption interfaces.IOWrapper = &ageIdentity

	entries := []struct {
		hash []byte
		data []byte
	}{
		{hash: sha256Hash([]byte("blob1")), data: []byte("hello encrypted world")},
		{hash: sha256Hash([]byte("blob2")), data: []byte("another encrypted blob")},
	}

	// Write with encryption
	writer, err := NewDataWriter(&buf, hashFormatId, ct, encryption)
	if err != nil {
		t.Fatal(err)
	}

	for _, e := range entries {
		if err := writer.WriteEntry(e.hash, e.data); err != nil {
			t.Fatal(err)
		}
	}

	_, writtenEntries, err := writer.Close()
	if err != nil {
		t.Fatal(err)
	}

	for i, we := range writtenEntries {
		if we.LogicalSize != uint64(len(entries[i].data)) {
			t.Errorf("entry %d: LogicalSize = %d, want %d",
				i, we.LogicalSize, len(entries[i].data))
		}
	}

	// Read with encryption — should recover plaintext
	reader, err := NewDataReader(bytes.NewReader(buf.Bytes()), encryption)
	if err != nil {
		t.Fatal(err)
	}

	readEntries, err := reader.ReadAllEntries()
	if err != nil {
		t.Fatal(err)
	}

	if len(readEntries) != len(entries) {
		t.Fatalf("got %d entries, want %d", len(readEntries), len(entries))
	}

	for i, re := range readEntries {
		if !bytes.Equal(re.Data, entries[i].data) {
			t.Errorf("entry %d: data mismatch", i)
		}
	}

	// Read WITHOUT encryption — data should not match plaintext
	readerNoKey, err := NewDataReader(bytes.NewReader(buf.Bytes()), nil)
	if err != nil {
		t.Fatal(err)
	}

	rawEntries, err := readerNoKey.ReadAllEntries()
	if err == nil && len(rawEntries) > 0 {
		if bytes.Equal(rawEntries[0].Data, entries[0].data) {
			t.Error("reading encrypted archive without key should not produce plaintext")
		}
	}
}

func sha256Hash(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}
