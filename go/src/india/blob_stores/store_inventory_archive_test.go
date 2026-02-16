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

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/bravo/markl_io"
	"code.linenisgreat.com/dodder/go/src/bravo/ohio"
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

type stubBlobStore struct {
	domain_interfaces.BlobStore
	makeBlobReaderCalled bool
	makeBlobReaderId     domain_interfaces.MarklId
}

func (s *stubBlobStore) MakeBlobReader(
	id domain_interfaces.MarklId,
) (domain_interfaces.BlobReader, error) {
	s.makeBlobReaderCalled = true
	s.makeBlobReaderId = id

	return markl_io.MakeNopReadCloser(
		markl.FormatHashSha256.Get(),
		ohio.NopCloser(bytes.NewReader(nil)),
	), nil
}

func (s *stubBlobStore) HasBlob(
	id domain_interfaces.MarklId,
) bool {
	return false
}

func TestMakeBlobReaderFallsBackToLoose(t *testing.T) {
	hashFormat := markl.FormatHashSha256

	stub := &stubBlobStore{}

	store := inventoryArchive{
		defaultHash:    hashFormat,
		basePath:       t.TempDir(),
		index:          make(map[string]archiveEntry),
		looseBlobStore: stub,
	}

	nonNullHash := sha256.Sum256([]byte("not in archive"))
	marklId, repool := hashFormat.GetBlobIdForHexString(
		hex.EncodeToString(nonNullHash[:]),
	)
	defer repool()

	reader, err := store.MakeBlobReader(marklId)
	if err != nil {
		t.Fatalf("MakeBlobReader: %v", err)
	}

	defer reader.Close()

	if !stub.makeBlobReaderCalled {
		t.Fatal("expected MakeBlobReader to delegate to loose blob store")
	}
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

	if !nullId.IsNull() {
		t.Fatal("test setup: expected null ID")
	}

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

func TestLoadIndexRebuildsFromIndexFiles(t *testing.T) {
	basePath := t.TempDir()
	cachePath := t.TempDir()

	hashFormatId := markl.FormatIdHashSha256
	hashFormat := markl.FormatHashSha256
	ct := compression_type.CompressionTypeNone

	testData := []byte("blob for index loading test")
	rawHash := sha256.Sum256(testData)

	// Write a data archive file
	var archiveBuf bytes.Buffer

	dataWriter, err := inventory_archive.NewDataWriter(
		&archiveBuf,
		hashFormatId,
		ct,
	)
	if err != nil {
		t.Fatalf("NewDataWriter: %v", err)
	}

	if err := dataWriter.WriteEntry(rawHash[:], testData); err != nil {
		t.Fatalf("WriteEntry: %v", err)
	}

	checksum, writtenEntries, err := dataWriter.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	archiveChecksum := hex.EncodeToString(checksum)
	archiveDataPath := filepath.Join(
		basePath,
		archiveChecksum+inventory_archive.DataFileExtension,
	)

	if err := os.WriteFile(
		archiveDataPath,
		archiveBuf.Bytes(),
		0o644,
	); err != nil {
		t.Fatalf("writing archive data file: %v", err)
	}

	// Write a corresponding index file
	indexEntries := []inventory_archive.IndexEntry{
		{
			Hash:           rawHash[:],
			PackOffset:     writtenEntries[0].Offset,
			CompressedSize: writtenEntries[0].CompressedSize,
		},
	}

	var indexBuf bytes.Buffer

	if _, err := inventory_archive.WriteIndex(
		&indexBuf,
		hashFormatId,
		indexEntries,
	); err != nil {
		t.Fatalf("WriteIndex: %v", err)
	}

	indexPath := filepath.Join(
		basePath,
		archiveChecksum+inventory_archive.IndexFileExtension,
	)

	if err := os.WriteFile(indexPath, indexBuf.Bytes(), 0o644); err != nil {
		t.Fatalf("writing index file: %v", err)
	}

	// Construct the store and let loadIndex rebuild from index files
	stub := &stubBlobStore{}

	store := inventoryArchive{
		defaultHash:    hashFormat,
		basePath:       basePath,
		cachePath:      cachePath,
		looseBlobStore: stub,
		index:          make(map[string]archiveEntry),
	}

	if err := store.loadIndex(); err != nil {
		t.Fatalf("loadIndex: %v", err)
	}

	// Verify HasBlob returns true for the archived blob
	marklId, repool := hashFormat.GetBlobIdForHexString(
		hex.EncodeToString(rawHash[:]),
	)
	defer repool()

	if !store.HasBlob(marklId) {
		t.Fatal("expected HasBlob to return true for archived blob")
	}

	// Verify the cache file was written
	cacheFilePath := filepath.Join(
		cachePath,
		inventory_archive.CacheFileName,
	)

	if _, err := os.Stat(cacheFilePath); err != nil {
		t.Fatalf("expected cache file to exist at %s: %v", cacheFilePath, err)
	}

	// Verify a second loadIndex uses the cache (no index files needed)
	store2 := inventoryArchive{
		defaultHash:    hashFormat,
		basePath:       t.TempDir(), // empty dir — no index files
		cachePath:      cachePath,
		looseBlobStore: stub,
		index:          make(map[string]archiveEntry),
	}

	if err := store2.loadIndex(); err != nil {
		t.Fatalf("loadIndex from cache: %v", err)
	}

	if !store2.HasBlob(marklId) {
		t.Fatal("expected HasBlob to return true when loaded from cache")
	}
}
