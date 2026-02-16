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

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/bravo/markl_io"
	"code.linenisgreat.com/dodder/go/src/bravo/ohio"
	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
	"code.linenisgreat.com/dodder/go/src/echo/inventory_archive"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/golf/blob_store_configs"
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
	allBlobIds           []domain_interfaces.MarklId
	blobData             map[string][]byte
}

func (s *stubBlobStore) MakeBlobReader(
	id domain_interfaces.MarklId,
) (domain_interfaces.BlobReader, error) {
	s.makeBlobReaderCalled = true
	s.makeBlobReaderId = id

	if s.blobData != nil {
		if data, ok := s.blobData[id.String()]; ok {
			return markl_io.MakeReadCloser(
				markl.FormatHashSha256.Get(),
				bytes.NewReader(data),
			), nil
		}
	}

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

func (s *stubBlobStore) AllBlobs() interfaces.SeqError[domain_interfaces.MarklId] {
	return func(yield func(domain_interfaces.MarklId, error) bool) {
		for _, id := range s.allBlobIds {
			if !yield(id, nil) {
				return
			}
		}
	}
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

func TestAllBlobsDeduplication(t *testing.T) {
	hashFormat := markl.FormatHashSha256

	// Create two hashes: one in archive index, one only in loose
	archiveHash := sha256.Sum256([]byte("archived blob"))
	looseOnlyHash := sha256.Sum256([]byte("loose only blob"))

	archiveId, archiveRepool := hashFormat.GetBlobIdForHexString(
		hex.EncodeToString(archiveHash[:]),
	)
	defer archiveRepool()

	looseOnlyId, looseOnlyRepool := hashFormat.GetBlobIdForHexString(
		hex.EncodeToString(looseOnlyHash[:]),
	)
	defer looseOnlyRepool()

	// The stub loose store returns both hashes
	stub := &stubBlobStore{
		allBlobIds: []domain_interfaces.MarklId{archiveId, looseOnlyId},
	}

	store := inventoryArchive{
		defaultHash:    hashFormat,
		basePath:       t.TempDir(),
		looseBlobStore: stub,
		index: map[string]archiveEntry{
			archiveId.String(): {
				ArchiveChecksum: "deadbeef",
				Offset:          0,
				CompressedSize:  100,
			},
		},
	}

	seen := make(map[string]int)

	for id, err := range store.AllBlobs() {
		if err != nil {
			t.Fatalf("AllBlobs error: %v", err)
		}

		seen[id.String()]++
	}

	// archiveId should appear exactly once (from archive, not from loose)
	if count := seen[archiveId.String()]; count != 1 {
		t.Errorf("archive blob seen %d times, want 1", count)
	}

	// looseOnlyId should appear exactly once (from loose)
	if count := seen[looseOnlyId.String()]; count != 1 {
		t.Errorf("loose-only blob seen %d times, want 1", count)
	}

	// Total should be exactly 2
	if len(seen) != 2 {
		t.Errorf("expected 2 unique blobs, got %d", len(seen))
	}
}

func TestPack(t *testing.T) {
	basePath := t.TempDir()
	cachePath := t.TempDir()

	hashFormat := markl.FormatHashSha256

	testData1 := []byte("pack test blob one")
	testData2 := []byte("pack test blob two")
	rawHash1 := sha256.Sum256(testData1)
	rawHash2 := sha256.Sum256(testData2)

	id1, repool1 := hashFormat.GetBlobIdForHexString(
		hex.EncodeToString(rawHash1[:]),
	)
	defer repool1()

	id2, repool2 := hashFormat.GetBlobIdForHexString(
		hex.EncodeToString(rawHash2[:]),
	)
	defer repool2()

	stub := &stubBlobStore{
		allBlobIds: []domain_interfaces.MarklId{id1, id2},
		blobData: map[string][]byte{
			id1.String(): testData1,
			id2.String(): testData2,
		},
	}

	config := blob_store_configs.TomlInventoryArchiveV0{
		HashTypeId:      markl.FormatIdHashSha256,
		CompressionType: compression_type.CompressionTypeNone,
	}

	store := inventoryArchive{
		defaultHash:    hashFormat,
		basePath:       basePath,
		cachePath:      cachePath,
		looseBlobStore: stub,
		index:          make(map[string]archiveEntry),
		config:         config,
	}

	if err := store.Pack(); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	// Verify both blobs are now in the in-memory index
	if !store.HasBlob(id1) {
		t.Fatal("expected id1 in index after pack")
	}

	if !store.HasBlob(id2) {
		t.Fatal("expected id2 in index after pack")
	}

	// Verify archive data file was written
	dataMatches, err := filepath.Glob(
		filepath.Join(basePath, "*"+inventory_archive.DataFileExtension),
	)
	if err != nil {
		t.Fatalf("globbing data files: %v", err)
	}

	if len(dataMatches) != 1 {
		t.Fatalf("expected 1 data file, got %d", len(dataMatches))
	}

	// Verify index file was written
	indexMatches, err := filepath.Glob(
		filepath.Join(basePath, "*"+inventory_archive.IndexFileExtension),
	)
	if err != nil {
		t.Fatalf("globbing index files: %v", err)
	}

	if len(indexMatches) != 1 {
		t.Fatalf("expected 1 index file, got %d", len(indexMatches))
	}

	// Verify cache file was written
	cacheFilePath := filepath.Join(cachePath, inventory_archive.CacheFileName)

	if _, err := os.Stat(cacheFilePath); err != nil {
		t.Fatalf("expected cache file at %s: %v", cacheFilePath, err)
	}

	// Verify we can read the packed blobs from the archive
	reader1, err := store.MakeBlobReader(id1)
	if err != nil {
		t.Fatalf("MakeBlobReader for id1 after pack: %v", err)
	}

	defer reader1.Close()

	got1, err := io.ReadAll(reader1)
	if err != nil {
		t.Fatalf("ReadAll for id1: %v", err)
	}

	if !bytes.Equal(got1, testData1) {
		t.Errorf("id1 data mismatch: got %q, want %q", got1, testData1)
	}

	reader2, err := store.MakeBlobReader(id2)
	if err != nil {
		t.Fatalf("MakeBlobReader for id2 after pack: %v", err)
	}

	defer reader2.Close()

	got2, err := io.ReadAll(reader2)
	if err != nil {
		t.Fatalf("ReadAll for id2: %v", err)
	}

	if !bytes.Equal(got2, testData2) {
		t.Errorf("id2 data mismatch: got %q, want %q", got2, testData2)
	}
}

func TestPackSkipsAlreadyArchivedBlobs(t *testing.T) {
	basePath := t.TempDir()
	cachePath := t.TempDir()

	hashFormat := markl.FormatHashSha256

	testData := []byte("already archived blob")
	rawHash := sha256.Sum256(testData)

	id, repool := hashFormat.GetBlobIdForHexString(
		hex.EncodeToString(rawHash[:]),
	)
	defer repool()

	stub := &stubBlobStore{
		allBlobIds: []domain_interfaces.MarklId{id},
		blobData: map[string][]byte{
			id.String(): testData,
		},
	}

	config := blob_store_configs.TomlInventoryArchiveV0{
		HashTypeId:      markl.FormatIdHashSha256,
		CompressionType: compression_type.CompressionTypeNone,
	}

	store := inventoryArchive{
		defaultHash:    hashFormat,
		basePath:       basePath,
		cachePath:      cachePath,
		looseBlobStore: stub,
		index: map[string]archiveEntry{
			id.String(): {
				ArchiveChecksum: "deadbeef00deadbeef00deadbeef00deadbeef00deadbeef00deadbeef00dead",
				Offset:          0,
				CompressedSize:  100,
			},
		},
		config: config,
	}

	if err := store.Pack(); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	// No new archive files should have been written since all blobs
	// were already in the index
	dataMatches, err := filepath.Glob(
		filepath.Join(basePath, "*"+inventory_archive.DataFileExtension),
	)
	if err != nil {
		t.Fatalf("globbing data files: %v", err)
	}

	if len(dataMatches) != 0 {
		t.Fatalf("expected 0 data files (all blobs already archived), got %d",
			len(dataMatches))
	}
}
