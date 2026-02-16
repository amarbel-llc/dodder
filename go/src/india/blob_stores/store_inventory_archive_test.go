//go:build test && debug

package blob_stores

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"

	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
	"code.linenisgreat.com/dodder/go/src/echo/inventory_archive"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
)

func TestMakeBlobReaderFromArchive(t *testing.T) {
	tmpDir := t.TempDir()

	hashFormatId := markl.FormatIdHashSha256
	ct := compression_type.CompressionTypeNone

	testData := []byte("hello from the archive")
	rawHash := sha256.Sum256(testData)

	// Write a data archive file
	var archiveBuf bytes.Buffer

	writer, err := inventory_archive.NewDataWriter(
		&archiveBuf,
		hashFormatId,
		ct,
	)
	if err != nil {
		t.Fatalf("NewDataWriter: %v", err)
	}

	if err := writer.WriteEntry(rawHash[:], testData); err != nil {
		t.Fatalf("WriteEntry: %v", err)
	}

	checksum, writtenEntries, err := writer.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	if len(writtenEntries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(writtenEntries))
	}

	// Write the archive file to disk
	archiveChecksum := hex.EncodeToString(checksum)
	archiveFileName := archiveChecksum + inventory_archive.DataFileExtension
	archivePath := filepath.Join(tmpDir, archiveFileName)

	if err := os.WriteFile(archivePath, archiveBuf.Bytes(), 0o644); err != nil {
		t.Fatalf("writing archive file: %v", err)
	}

	// Build a markl ID for the blob hash
	hashFormat := markl.FormatHashSha256
	marklId, repool := hashFormat.GetBlobIdForHexString(
		hex.EncodeToString(rawHash[:]),
	)
	defer repool()

	// Build the store directly with a pre-populated index
	store := inventoryArchive{
		defaultHash: hashFormat,
		basePath:    tmpDir,
		index: map[string]archiveEntry{
			marklId.String(): {
				ArchiveChecksum: archiveChecksum,
				Offset:          writtenEntries[0].Offset,
				CompressedSize:  writtenEntries[0].CompressedSize,
			},
		},
	}

	// Read the blob
	reader, err := store.MakeBlobReader(marklId)
	if err != nil {
		t.Fatalf("MakeBlobReader: %v", err)
	}

	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if !bytes.Equal(got, testData) {
		t.Errorf("data mismatch: got %q, want %q", got, testData)
	}
}

func TestMakeBlobReaderFallsBackToLoose(t *testing.T) {
	hashFormat := markl.FormatHashSha256

	// Build a store with empty index and nil loose store
	// When ID is not in the index, it should delegate to looseBlobStore.
	// We don't have a real loose store here, so just verify the delegation path
	// by checking that a missing entry doesn't panic and calls through.
	store := inventoryArchive{
		defaultHash: hashFormat,
		basePath:    t.TempDir(),
		index:       make(map[string]archiveEntry),
	}

	// Create a non-null ID that is NOT in the index
	nonNullHash := sha256.Sum256([]byte("not in archive"))
	marklId, repool := hashFormat.GetBlobIdForHexString(
		hex.EncodeToString(nonNullHash[:]),
	)
	defer repool()

	// Without a looseBlobStore, this should panic/fail, confirming
	// that MakeBlobReader delegates to loose when not in index.
	// We verify by checking it doesn't try to open an archive file.
	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic from nil looseBlobStore delegation")
			}
		}()

		store.MakeBlobReader(marklId)
	}()
}

func TestMakeBlobReaderNullIdReturnsNopReader(t *testing.T) {
	hashFormat := markl.FormatHashSha256

	store := inventoryArchive{
		defaultHash: hashFormat,
		basePath:    t.TempDir(),
		index:       make(map[string]archiveEntry),
	}

	// A null markl ID (zero hash) should return an empty reader
	nullId := hashFormat.FromStringContent("")

	reader, err := store.MakeBlobReader(nullId)
	if err != nil {
		t.Fatalf("MakeBlobReader for null: %v", err)
	}

	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("expected empty data for null id, got %d bytes", len(got))
	}
}

func TestMakeBlobReaderFromArchiveZstd(t *testing.T) {
	tmpDir := t.TempDir()

	hashFormatId := markl.FormatIdHashSha256
	ct := compression_type.CompressionTypeZstd

	testData := []byte("compressed archive data that repeats repeats repeats")
	rawHash := sha256.Sum256(testData)

	var archiveBuf bytes.Buffer

	writer, err := inventory_archive.NewDataWriter(
		&archiveBuf,
		hashFormatId,
		ct,
	)
	if err != nil {
		t.Fatalf("NewDataWriter: %v", err)
	}

	if err := writer.WriteEntry(rawHash[:], testData); err != nil {
		t.Fatalf("WriteEntry: %v", err)
	}

	checksum, writtenEntries, err := writer.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	archiveChecksum := hex.EncodeToString(checksum)
	archiveFileName := archiveChecksum + inventory_archive.DataFileExtension
	archivePath := filepath.Join(tmpDir, archiveFileName)

	if err := os.WriteFile(archivePath, archiveBuf.Bytes(), 0o644); err != nil {
		t.Fatalf("writing archive file: %v", err)
	}

	hashFormat := markl.FormatHashSha256
	marklId, repool := hashFormat.GetBlobIdForHexString(
		hex.EncodeToString(rawHash[:]),
	)
	defer repool()

	store := inventoryArchive{
		defaultHash: hashFormat,
		basePath:    tmpDir,
		index: map[string]archiveEntry{
			marklId.String(): {
				ArchiveChecksum: archiveChecksum,
				Offset:          writtenEntries[0].Offset,
				CompressedSize:  writtenEntries[0].CompressedSize,
			},
		},
	}

	reader, err := store.MakeBlobReader(marklId)
	if err != nil {
		t.Fatalf("MakeBlobReader: %v", err)
	}

	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	if !bytes.Equal(got, testData) {
		t.Errorf("data mismatch: got %q, want %q", got, testData)
	}
}
