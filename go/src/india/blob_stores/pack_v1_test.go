//go:build test && debug

package blob_stores

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
	"code.linenisgreat.com/dodder/go/src/echo/inventory_archive"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/golf/blob_store_configs"
)

func TestPackV1WithDelta(t *testing.T) {
	basePath := t.TempDir()
	cachePath := t.TempDir()

	hashFormat := markl.FormatHashSha256

	// Create similar blobs that should benefit from delta encoding.
	// The blobs share a large common prefix to produce small deltas.
	commonPrefix := strings.Repeat("shared content block ", 100)
	testData1 := []byte(commonPrefix + " unique suffix alpha")
	testData2 := []byte(commonPrefix + " unique suffix beta")
	testData3 := []byte(commonPrefix + " unique suffix gamma")

	rawHash1 := sha256.Sum256(testData1)
	rawHash2 := sha256.Sum256(testData2)
	rawHash3 := sha256.Sum256(testData3)

	id1, repool1 := hashFormat.GetBlobIdForHexString(
		hex.EncodeToString(rawHash1[:]),
	)
	defer repool1()

	id2, repool2 := hashFormat.GetBlobIdForHexString(
		hex.EncodeToString(rawHash2[:]),
	)
	defer repool2()

	id3, repool3 := hashFormat.GetBlobIdForHexString(
		hex.EncodeToString(rawHash3[:]),
	)
	defer repool3()

	stub := &stubBlobStore{
		allBlobIds: []domain_interfaces.MarklId{id1, id2, id3},
		blobData: map[string][]byte{
			id1.String(): testData1,
			id2.String(): testData2,
			id3.String(): testData3,
		},
	}

	config := blob_store_configs.TomlInventoryArchiveV1{
		HashTypeId:      markl.FormatIdHashSha256,
		CompressionType: compression_type.CompressionTypeNone,
		Delta: blob_store_configs.DeltaConfig{
			Enabled:     true,
			Algorithm:   "bsdiff",
			MinBlobSize: 1,
			MaxBlobSize: 10485760,
			SizeRatio:   2.0,
		},
	}

	store := inventoryArchiveV1{
		defaultHash:    hashFormat,
		basePath:       basePath,
		cachePath:      cachePath,
		looseBlobStore: stub,
		index:          make(map[string]archiveEntryV1),
		config:         config,
	}

	if err := store.Pack(PackOptions{}); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	// Verify all blobs are in the in-memory index.
	if !store.HasBlob(id1) {
		t.Fatal("expected id1 in index after pack")
	}

	if !store.HasBlob(id2) {
		t.Fatal("expected id2 in index after pack")
	}

	if !store.HasBlob(id3) {
		t.Fatal("expected id3 in index after pack")
	}

	// Verify at least one entry was stored as delta.
	deltaCount := 0
	for _, entry := range store.index {
		if entry.EntryType == inventory_archive.EntryTypeDelta {
			deltaCount++
		}
	}

	if deltaCount == 0 {
		t.Fatal("expected at least one delta entry, got none")
	}

	t.Logf("delta entries: %d, total entries: %d", deltaCount, len(store.index))

	// Verify all blobs are readable and produce the correct data.
	for _, tc := range []struct {
		name string
		id   domain_interfaces.MarklId
		data []byte
	}{
		{"blob1", id1, testData1},
		{"blob2", id2, testData2},
		{"blob3", id3, testData3},
	} {
		reader, err := store.MakeBlobReader(tc.id)
		if err != nil {
			t.Fatalf("MakeBlobReader for %s: %v", tc.name, err)
		}

		got, err := io.ReadAll(reader)
		reader.Close()

		if err != nil {
			t.Fatalf("ReadAll for %s: %v", tc.name, err)
		}

		if !bytes.Equal(got, tc.data) {
			t.Errorf(
				"%s data mismatch: got %d bytes, want %d bytes",
				tc.name,
				len(got),
				len(tc.data),
			)
		}
	}

	// Verify v1 data file was written.
	dataMatches, err := filepath.Glob(
		filepath.Join(basePath, "*"+inventory_archive.DataFileExtensionV1),
	)
	if err != nil {
		t.Fatalf("globbing data files: %v", err)
	}

	if len(dataMatches) != 1 {
		t.Fatalf("expected 1 v1 data file, got %d", len(dataMatches))
	}

	// Verify v1 index file was written.
	indexMatches, err := filepath.Glob(
		filepath.Join(basePath, "*"+inventory_archive.IndexFileExtensionV1),
	)
	if err != nil {
		t.Fatalf("globbing index files: %v", err)
	}

	if len(indexMatches) != 1 {
		t.Fatalf("expected 1 v1 index file, got %d", len(indexMatches))
	}

	// Verify v1 cache file was written.
	cacheFilePath := filepath.Join(
		cachePath,
		inventory_archive.CacheFileNameV1,
	)

	if _, err := os.Stat(cacheFilePath); err != nil {
		t.Fatalf("expected v1 cache file at %s: %v", cacheFilePath, err)
	}
}

func TestPackV1WithoutDelta(t *testing.T) {
	basePath := t.TempDir()
	cachePath := t.TempDir()

	hashFormat := markl.FormatHashSha256

	testData1 := []byte("pack v1 no-delta blob one")
	testData2 := []byte("pack v1 no-delta blob two")

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

	config := blob_store_configs.TomlInventoryArchiveV1{
		HashTypeId:      markl.FormatIdHashSha256,
		CompressionType: compression_type.CompressionTypeNone,
		Delta: blob_store_configs.DeltaConfig{
			Enabled:     false,
			Algorithm:   "bsdiff",
			MinBlobSize: 1,
			MaxBlobSize: 10485760,
			SizeRatio:   2.0,
		},
	}

	store := inventoryArchiveV1{
		defaultHash:    hashFormat,
		basePath:       basePath,
		cachePath:      cachePath,
		looseBlobStore: stub,
		index:          make(map[string]archiveEntryV1),
		config:         config,
	}

	if err := store.Pack(PackOptions{}); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	// Verify all blobs are in the index.
	if !store.HasBlob(id1) {
		t.Fatal("expected id1 in index after pack")
	}

	if !store.HasBlob(id2) {
		t.Fatal("expected id2 in index after pack")
	}

	// Verify all entries are full (no deltas).
	for key, entry := range store.index {
		if entry.EntryType != inventory_archive.EntryTypeFull {
			t.Errorf(
				"expected full entry for %s, got entry type %d",
				key,
				entry.EntryType,
			)
		}
	}

	// Verify all blobs are readable.
	for _, tc := range []struct {
		name string
		id   domain_interfaces.MarklId
		data []byte
	}{
		{"blob1", id1, testData1},
		{"blob2", id2, testData2},
	} {
		reader, err := store.MakeBlobReader(tc.id)
		if err != nil {
			t.Fatalf("MakeBlobReader for %s: %v", tc.name, err)
		}

		got, err := io.ReadAll(reader)
		reader.Close()

		if err != nil {
			t.Fatalf("ReadAll for %s: %v", tc.name, err)
		}

		if !bytes.Equal(got, tc.data) {
			t.Errorf("%s data mismatch", tc.name)
		}
	}
}

func TestPackV1DeltaFallsBackToFullWhenLarger(t *testing.T) {
	basePath := t.TempDir()
	cachePath := t.TempDir()

	hashFormat := markl.FormatHashSha256

	// Create blobs with truly random content so deltas are guaranteed to be
	// larger than the original data, triggering the trial-and-discard fallback.
	var blobDatas [][]byte
	var blobIds []domain_interfaces.MarklId

	blobData := make(map[string][]byte)

	for i := range 3 {
		// Random data is incompressible and produces deltas larger than
		// the original. Use similar sizes so the selector still pairs them.
		data := make([]byte, 2048+i*100)
		if _, randErr := rand.Read(data); randErr != nil {
			t.Fatalf("crypto/rand.Read: %v", randErr)
		}

		blobDatas = append(blobDatas, data)

		rawHash := sha256.Sum256(data)
		id, repool := hashFormat.GetBlobIdForHexString(
			hex.EncodeToString(rawHash[:]),
		)
		defer repool()

		blobIds = append(blobIds, id)
		blobData[id.String()] = data
	}

	stub := &stubBlobStore{
		allBlobIds: blobIds,
		blobData:   blobData,
	}

	config := blob_store_configs.TomlInventoryArchiveV1{
		HashTypeId:      markl.FormatIdHashSha256,
		CompressionType: compression_type.CompressionTypeNone,
		Delta: blob_store_configs.DeltaConfig{
			Enabled:     true,
			Algorithm:   "bsdiff",
			MinBlobSize: 1,
			MaxBlobSize: 10485760,
			SizeRatio:   2.0,
		},
	}

	store := inventoryArchiveV1{
		defaultHash:    hashFormat,
		basePath:       basePath,
		cachePath:      cachePath,
		looseBlobStore: stub,
		index:          make(map[string]archiveEntryV1),
		config:         config,
	}

	if err := store.Pack(PackOptions{}); err != nil {
		t.Fatalf("Pack: %v", err)
	}

	// Verify all blobs are in the index.
	for i, id := range blobIds {
		if !store.HasBlob(id) {
			t.Fatalf("expected blob %d in index after pack", i)
		}
	}

	// Verify all blobs are readable and produce correct data.
	for i, id := range blobIds {
		reader, err := store.MakeBlobReader(id)
		if err != nil {
			t.Fatalf("MakeBlobReader for blob %d: %v", i, err)
		}

		got, err := io.ReadAll(reader)
		reader.Close()

		if err != nil {
			t.Fatalf("ReadAll for blob %d: %v", i, err)
		}

		if !bytes.Equal(got, blobDatas[i]) {
			t.Errorf(
				"blob %d data mismatch: got %d bytes, want %d bytes",
				i,
				len(got),
				len(blobDatas[i]),
			)
		}
	}

	// All entries must be stored as full since the random content produces
	// deltas larger than the original data.
	fullCount := 0
	for _, entry := range store.index {
		if entry.EntryType == inventory_archive.EntryTypeFull {
			fullCount++
		}
	}

	if fullCount != len(store.index) {
		t.Errorf(
			"expected all entries to be full, got %d full out of %d",
			fullCount,
			len(store.index),
		)
	}
}
