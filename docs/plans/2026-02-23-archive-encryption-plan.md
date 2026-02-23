# Archive Encryption Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add per-entry encryption to inventory archive data files and rename size fields to git-style `StoredSize`/`LogicalSize`.

**Architecture:** Two phases. Phase 1 renames struct fields, local variables, function parameters, and wire format comments from compression-centric names to storage-centric names. Phase 2 adds an `IOWrapper` encryption parameter to data writers/readers, wires it through the pack and read paths in `blob_stores`, and adds tests.

**Tech Stack:** Go, `interfaces.IOWrapper`, `delta/age` (X25519), `inventory_archive` binary format, BATS integration tests.

**Design doc:** `docs/plans/2026-02-23-archive-encryption-design.md`

---

## Phase 1: Field and Variable Renames

### Task 1: Rename struct fields in types.go

**Files:**
- Modify: `go/src/echo/inventory_archive/types.go`

**Step 1: Rename all struct fields**

In `types.go`, make these changes across all six structs:

```go
// DataEntry (line 48-55)
type DataEntry struct {
	Hash       []byte
	LogicalSize  uint64   // was UncompressedSize
	StoredSize   uint64   // was CompressedSize
	Data       []byte
	Offset     uint64
}

// IndexEntry (line 56-60)
type IndexEntry struct {
	Hash       []byte
	PackOffset uint64
	StoredSize uint64   // was CompressedSize
}

// CacheEntry (line 62-67)
type CacheEntry struct {
	Hash            []byte
	ArchiveChecksum []byte
	Offset          uint64
	StoredSize      uint64   // was CompressedSize
}

// DataEntryV1 (line 69-80)
type DataEntryV1 struct {
	Hash           []byte
	EntryType      byte
	Encoding       byte
	LogicalSize    uint64   // was UncompressedSize
	StoredSize     uint64   // was CompressedSize
	Data           []byte
	Offset         uint64
	DeltaAlgorithm byte
	BaseHash       []byte
}

// IndexEntryV1 (line 82-88)
type IndexEntryV1 struct {
	Hash       []byte
	PackOffset uint64
	StoredSize uint64   // was CompressedSize
	EntryType  byte
	BaseOffset uint64
}

// CacheEntryV1 (line 90-97)
type CacheEntryV1 struct {
	Hash            []byte
	ArchiveChecksum []byte
	Offset          uint64
	StoredSize      uint64   // was CompressedSize
	EntryType       byte
	BaseOffset      uint64
}
```

Also update the comment on `DataEntryV1.StoredSize`:
```go
StoredSize     uint64 // For delta entries, this is the stored delta payload size
```

**Step 2: Fix all compilation errors**

Run: `cd /home/sasha/eng/repos/dodder/go && go build ./...`

This will produce errors everywhere `CompressedSize` and `UncompressedSize` are
referenced. Fix each one mechanically:

- `CompressedSize` → `StoredSize` everywhere
- `UncompressedSize` → `LogicalSize` everywhere

Files that will need updating (every reference):

`echo/inventory_archive/`:
- `data_writer.go`: lines 196-197, 206-208
- `data_reader.go`: lines 167, 177, 184
- `data_writer_v1.go`: lines 222-223, 234-236, 340-341, 355-357
- `data_reader_v1.go`: lines 220, 230, 237, 253, 293, 303, 310
- `index_writer.go`: line 195
- `index_reader.go`: lines 200, 209, 245
- `index_v1.go`: lines 195, 413, 430, 468
- `cache_writer.go`: line 179
- `cache_reader.go`: line 178
- `cache_v1.go`: lines 180, 372
- `data_writer_test.go`: lines 69, 73, 127, 131, 242, 246
- `data_writer_v1_test.go`: lines 70, 74, 124, 128, 273, 276-277
- `index_test.go`: lines 22, 105, 109-110, 159, 164
- `index_v1_test.go`: lines 22, 113, 117-118, 185, 190
- `cache_test.go`: lines 28, 115, 119-120, 182, 186-187
- `cache_v1_test.go`: lines 28, 123, 127-128, 208, 212-213

`india/blob_stores/`:
- `store_inventory_archive.go`: lines 25, 86, 201, 210
- `store_inventory_archive_v1.go`: lines 25, 88, 205, 216
- `pack_v0.go`: lines 325, 363, 395
- `pack_v1.go`: lines 476, 531, 565
- `store_inventory_archive_test.go`: lines 82, 273, 346, 452, 639

**Step 3: Verify compilation**

Run: `cd /home/sasha/eng/repos/dodder/go && go build ./...`
Expected: Clean build, no errors.

**Step 4: Run tests**

Run: `cd /home/sasha/eng/repos/dodder/go && go test -v -tags test,debug ./src/echo/inventory_archive/... ./src/india/blob_stores/...`
Expected: All tests pass.

**Step 5: Commit**

```
git add -A && git commit -m "refactor: rename CompressedSize->StoredSize, UncompressedSize->LogicalSize"
```

---

### Task 2: Rename local variables and function parameters

**Files:**
- Modify: `go/src/echo/inventory_archive/data_writer.go`
- Modify: `go/src/echo/inventory_archive/data_reader.go`
- Modify: `go/src/echo/inventory_archive/data_writer_v1.go`
- Modify: `go/src/echo/inventory_archive/data_reader_v1.go`
- Modify: `go/src/echo/inventory_archive/index_reader.go`
- Modify: `go/src/echo/inventory_archive/index_v1.go`

**Step 1: Rename local variables in data writers and readers**

In `data_writer.go` `WriteEntry()`:
- `uncompressedSize` → `logicalSize` (line 145)
- `compressedSize` → `storedSize` (line 176)
- `compressedData` → keep as-is (it describes the compression output accurately)
- Comments: `// Write uncompressed_size` → `// Write logical_size`
- Comments: `// Write compressed_size` → `// Write stored_size`
- Comments: `// Write compressed data` → `// Write payload`
- Offset comment: `8 + // uncompressed_size` → `8 + // logical_size`
- Offset comment: `8 + // compressed_size` → `8 + // stored_size`
- Offset comment: `compressedSize // data` → `storedSize // payload`

In `data_reader.go` `ReadEntry()`:
- Comments: `// Read uncompressed_size` → `// Read logical_size`
- Comments: `// Read compressed_size` → `// Read stored_size`
- `compressedData` → `storedData` (line 184, since on read this is what was stored on disk)
- Comments: `// Read compressed data` → `// Read payload`
- Error message: `"reading compressed data"` → `"reading payload"`

In `data_writer_v1.go` `WriteFullEntry()`:
- `uncompressedSize` → `logicalSize` (line 169)
- `compressedSize` → `storedSize` (line 190)
- Comments: `// uncompressed_size` → `// logical_size`
- Comments: `// compressed_size` → `// stored_size`
- Comments: `// compressed_data` → `// payload`
- Offset comments: same pattern as V0

In `data_writer_v1.go` `WriteDeltaEntry()`:
- Parameter: `uncompressedSize uint64` → `logicalSize uint64` (line 245)
- `deltaSize` → `storedSize` (line 306)
- Comments: `// uncompressed_size` → `// logical_size`
- Comments: `// delta_size` → `// stored_size`
- Comments: `// delta_data` → `// payload`
- Offset comments: `8 + // uncompressed_size` → `8 + // logical_size`
- Offset comments: `8 + // delta_size` → `8 + // stored_size`
- Offset comments: `deltaSize // delta_data` → `storedSize // payload`

In `data_reader_v1.go` `readFullEntryBody()`:
- Comments: `// uncompressed_size` → `// logical_size`
- Comments: `// compressed_size` → `// stored_size`
- `compressedData` → `storedData` (line 237)
- Comments: `// compressed_data` → `// payload`
- Error messages: `"reading compressed data"` → `"reading payload"`

In `data_reader_v1.go` `readDeltaEntryBody()`:
- Comments: `// uncompressed_size` → `// logical_size`
- Comments: `// delta_size` → `// stored_size`
- `compressedDelta` → `storedData` (line 310)
- Comments: `// delta_data (compressed)` → `// payload`
- Error messages: `"reading delta data"` → `"reading payload"`

In `index_reader.go` `LookupHash()`:
- Return parameter: `compressedSize uint64` → `storedSize uint64` (line 209)

In `index_v1.go` `LookupHash()`:
- Return parameter: `compressedSize uint64` → `storedSize uint64` (line 430)

**Step 2: Fix callers of LookupHash**

In test files that call `LookupHash` and capture `compressedSize`:

- `index_test.go` line 138: `packOffset, compressedSize, found, lookupErr` → `packOffset, storedSize, found, lookupErr`
- `index_test.go` line 159: `compressedSize` → `storedSize`
- `index_test.go` line 163: format string reference
- `index_v1_test.go` line 164: same pattern
- `index_v1_test.go` line 185: same pattern
- `index_v1_test.go` line 189: same pattern

Also fix the caller of `WriteDeltaEntry` in `india/blob_stores/pack_v1.go`:
- The call site passes a local var for `uncompressedSize` that matches the
  parameter name. Confirm the argument is still correct after parameter rename
  (it will be, since the value doesn't change — only the name).

**Step 3: Verify compilation and tests**

Run: `cd /home/sasha/eng/repos/dodder/go && go build ./... && go test -v -tags test,debug ./src/echo/inventory_archive/... ./src/india/blob_stores/...`
Expected: Clean build, all tests pass.

**Step 4: Commit**

```
git add -A && git commit -m "refactor: rename local vars and comments to stored_size/logical_size"
```

---

### Task 3: Rename in blob_stores layer

**Files:**
- Modify: `go/src/india/blob_stores/store_inventory_archive.go`
- Modify: `go/src/india/blob_stores/store_inventory_archive_v1.go`

**Step 1: Rename archiveEntry and archiveEntryV1 fields**

In `store_inventory_archive.go` (around line 22-26):
```go
type archiveEntry struct {
	ArchiveChecksum string
	Offset          uint64
	StoredSize      uint64   // was CompressedSize
}
```

In `store_inventory_archive_v1.go` (around line 22-27):
```go
type archiveEntryV1 struct {
	ArchiveChecksum string
	Offset          uint64
	StoredSize      uint64   // was CompressedSize
	EntryType       byte
	BaseOffset      uint64
}
```

Fix all references in both files and their test file.

**Step 2: Verify compilation and tests**

Run: `cd /home/sasha/eng/repos/dodder/go && go build ./... && go test -v -tags test,debug ./src/india/blob_stores/...`
Expected: Clean build, all tests pass.

**Step 3: Commit**

```
git add -A && git commit -m "refactor: rename archiveEntry.CompressedSize->StoredSize"
```

---

### Task 4: Run full test suite

**Step 1: Run unit tests**

Run: `cd /home/sasha/eng/repos/dodder/go && go test -v -tags test,debug ./...`
Expected: All pass.

**Step 2: Run BATS integration tests**

Run: `cd /home/sasha/eng/repos/dodder && /home/sasha/eng/result/bin/just test-bats`
Expected: All pass (314+ tests).

**Step 3: Format**

Run: `cd /home/sasha/eng/repos/dodder && /home/sasha/eng/result/bin/just codemod-go-fmt`

If formatter changed anything, amend the previous commit or make a new commit.

---

## Phase 2: Add Encryption Support

### Task 5: Add FlagHasEncryption constant and IOWrapper import

**Files:**
- Modify: `go/src/echo/inventory_archive/types.go`

**Step 1: Add constant**

In `types.go`, add to the V0 constants block:
```go
const (
	// ... existing V0 constants ...
	FlagHasEncryption uint16 = 1 << 0
)
```

And update the V1 constants block:
```go
const (
	// ... existing V1 constants ...
	FlagHasEncryptionV1 uint16 = 1 << 2
)
```

**Step 2: Verify compilation**

Run: `cd /home/sasha/eng/repos/dodder/go && go build ./src/echo/inventory_archive/...`
Expected: Clean build.

**Step 3: Commit**

```
git add -A && git commit -m "feat: add FlagHasEncryption constants for archive format"
```

---

### Task 6: Add encryption to V0 DataWriter

**Files:**
- Modify: `go/src/echo/inventory_archive/data_writer.go`
- Modify: `go/src/echo/inventory_archive/data_writer_test.go`

**Step 1: Write the failing test**

Add to `data_writer_test.go`:

```go
func TestDataWriterEncryptedRoundTrip(t *testing.T) {
	hashFormatId := "sha256"
	ct := compression_type.CompressionTypeZstd

	// Use age X25519 for encryption
	ageIdentity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}

	var ioWrapper interfaces.IOWrapper = ageIdentity

	entries := []struct {
		hash []byte
		data []byte
	}{
		{hash: sha256Hash("blob1"), data: []byte("hello encrypted world")},
		{hash: sha256Hash("blob2"), data: []byte("another encrypted blob")},
	}

	// Write with encryption
	var buf bytes.Buffer
	writer, err := NewDataWriter(&buf, hashFormatId, ct, ioWrapper)
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

	// Verify stored size differs from raw compressed size (encryption adds overhead)
	for i, we := range writtenEntries {
		if we.LogicalSize != uint64(len(entries[i].data)) {
			t.Errorf("entry %d: LogicalSize = %d, want %d",
				i, we.LogicalSize, len(entries[i].data))
		}
	}

	// Read with encryption
	reader, err := NewDataReader(bytes.NewReader(buf.Bytes()), ioWrapper)
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

	// Verify reading WITHOUT encryption fails to produce valid data
	readerNoKey, err := NewDataReader(bytes.NewReader(buf.Bytes()), nil)
	if err != nil {
		t.Fatal(err)
	}

	rawEntries, err := readerNoKey.ReadAllEntries()
	if err == nil && len(rawEntries) > 0 {
		// Data should not match plaintext (it's encrypted)
		if bytes.Equal(rawEntries[0].Data, entries[0].data) {
			t.Error("reading encrypted archive without key should not produce plaintext")
		}
	}
}
```

Note: you will need to add the `age` and `interfaces` imports to the test file.
The `sha256Hash` helper should already exist in the test file — check and reuse it.
If `NewDataWriter` and `NewDataReader` don't accept the IOWrapper parameter yet,
the test won't compile. That's the expected failure.

**Step 2: Run test to verify it fails**

Run: `cd /home/sasha/eng/repos/dodder/go && go test -v -tags test,debug -run TestDataWriterEncryptedRoundTrip ./src/echo/inventory_archive/...`
Expected: Compilation error — `NewDataWriter` signature mismatch.

**Step 3: Implement encryption in DataWriter**

In `data_writer.go`:

Add `interfaces` import:
```go
"code.linenisgreat.com/dodder/go/src/_/interfaces"
```

Add field to `DataWriter`:
```go
type DataWriter struct {
	// ... existing fields ...
	encryption interfaces.IOWrapper
}
```

Update `NewDataWriter` signature:
```go
func NewDataWriter(
	w io.Writer,
	hashFormatId string,
	ct compression_type.CompressionType,
	encryption interfaces.IOWrapper,
) (dw *DataWriter, err error) {
```

Store encryption in the struct and set flag in header if non-nil:
```go
	dw = &DataWriter{
		// ... existing fields ...
		encryption: encryption,
	}
```

In `writeHeader()`, update the flags bytes:
```go
	// flags: 2 bytes
	var flags uint16
	if dw.encryption != nil {
		flags |= FlagHasEncryption
	}

	if err = binary.Write(
		dw.multiWriter,
		binary.BigEndian,
		flags,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}
```

In `WriteEntry()`, after compression and before writing the payload, add
encryption:
```go
	compressedData := compressedBuf.Bytes()

	// Encrypt if configured
	storedData := compressedData
	if dw.encryption != nil {
		var encryptedBuf bytes.Buffer

		encryptWriter, encErr := dw.encryption.WrapWriter(&encryptedBuf)
		if encErr != nil {
			err = errors.Wrap(encErr)
			return err
		}

		if _, encErr = encryptWriter.Write(compressedData); encErr != nil {
			err = errors.Wrap(encErr)
			return err
		}

		if encErr = encryptWriter.Close(); encErr != nil {
			err = errors.Wrap(encErr)
			return err
		}

		storedData = encryptedBuf.Bytes()
	}

	storedSize := uint64(len(storedData))

	// Write stored_size
	// ...binary.Write storedSize...

	// Write payload
	// ...dw.multiWriter.Write(storedData)...
```

Update the entry struct and offset calculation to use `storedSize` and
`storedData`.

**Step 4: Implement encryption in DataReader**

In `data_reader.go`:

Add `interfaces` import.

Add field to `DataReader`:
```go
type DataReader struct {
	// ... existing fields ...
	encryption interfaces.IOWrapper
}
```

Update `NewDataReader` signature:
```go
func NewDataReader(
	r io.ReadSeeker,
	encryption interfaces.IOWrapper,
) (dr *DataReader, err error) {
```

Store encryption. In `readHeader()`, validate the flag:
```go
	var flags uint16

	if err = binary.Read(
		dr.reader,
		binary.BigEndian,
		&flags,
	); err != nil {
		err = errors.Wrapf(err, "reading flags")
		return err
	}

	if flags&FlagHasEncryption != 0 && dr.encryption == nil {
		// Archive is encrypted but no key provided — allow reading metadata
		// but data will be raw encrypted bytes
	}
```

In `ReadEntry()`, after reading stored payload and before decompression, add
decryption:
```go
	storedData := make([]byte, entry.StoredSize)

	if _, err = io.ReadFull(dr.reader, storedData); err != nil {
		err = errors.Wrapf(err, "reading payload")
		return entry, err
	}

	// Decrypt if needed
	dataToDecompress := storedData
	if dr.encryption != nil {
		decryptReader, decErr := dr.encryption.WrapReader(
			bytes.NewReader(storedData),
		)
		if decErr != nil {
			err = errors.Wrapf(decErr, "creating decryption reader")
			return entry, err
		}

		dataToDecompress, err = io.ReadAll(decryptReader)
		if err != nil {
			err = errors.Wrapf(err, "decrypting payload")
			return entry, err
		}

		if err = decryptReader.Close(); err != nil {
			err = errors.Wrapf(err, "closing decryption reader")
			return entry, err
		}
	}

	// Decompress
	decompressReader, err := dr.compressionType.WrapReader(
		bytes.NewReader(dataToDecompress),
	)
```

**Step 5: Fix all callers of NewDataWriter and NewDataReader**

Every existing call site needs `nil` appended for the new encryption parameter.
Search and fix:

- `data_writer_test.go`: all `NewDataWriter(` calls → add `, nil`
- `data_reader.go` is not called directly (NewDataReader is), but reader test
  uses `NewDataReader` implicitly through the writer test
- `india/blob_stores/pack_v0.go` `packChunkArchive()`: add `, nil` for now
  (Task 9 will wire the real IOWrapper)
- `india/blob_stores/pack_v0.go` `validateArchive()`: add `, nil`
- `india/blob_stores/store_inventory_archive.go` `MakeBlobReader()`: add `, nil`
- `india/blob_stores/store_inventory_archive.go` in index rebuild: add `, nil`
- `india/blob_stores/store_inventory_archive_test.go`: all calls → add `, nil`

**Step 6: Run tests**

Run: `cd /home/sasha/eng/repos/dodder/go && go test -v -tags test,debug ./src/echo/inventory_archive/... ./src/india/blob_stores/...`
Expected: All tests pass, including the new encrypted round-trip test.

**Step 7: Commit**

```
git add -A && git commit -m "feat: add encryption support to V0 archive data writer/reader"
```

---

### Task 7: Add encryption to V1 DataWriter and DataReader

**Files:**
- Modify: `go/src/echo/inventory_archive/data_writer_v1.go`
- Modify: `go/src/echo/inventory_archive/data_reader_v1.go`
- Modify: `go/src/echo/inventory_archive/data_writer_v1_test.go`

**Step 1: Write the failing test**

Add `TestDataWriterV1EncryptedRoundTrip` to `data_writer_v1_test.go`, similar
to the V0 test but using `NewDataWriterV1` and `NewDataReaderV1`. Include both
full and delta entries if the test infrastructure supports it.

**Step 2: Run test to verify it fails**

Expected: Compilation error — `NewDataWriterV1` signature mismatch.

**Step 3: Implement**

Same pattern as Task 6 but for V1:

- Add `encryption interfaces.IOWrapper` field to `DataWriterV1` and `DataReaderV1`
- Update `NewDataWriterV1` and `NewDataReaderV1` signatures
- In `writeHeader()`: set `FlagHasEncryptionV1` in flags if encryption non-nil
- In `WriteFullEntry()`: encrypt after compress, same pattern as V0
- In `WriteDeltaEntry()`: encrypt after compress, same pattern
- In `readFullEntryBody()`: decrypt before decompress
- In `readDeltaEntryBody()`: decrypt before decompress
- In `readHeader()`: read and validate `FlagHasEncryptionV1`

**Step 4: Fix all callers**

Add `, nil` to all existing `NewDataWriterV1` and `NewDataReaderV1` calls:

- `data_writer_v1_test.go`: all calls
- `india/blob_stores/pack_v1.go` `packChunkArchiveV1()`
- `india/blob_stores/store_inventory_archive_v1.go` `MakeBlobReader()`
- `india/blob_stores/store_inventory_archive_v1.go` index rebuild
- `india/blob_stores/pack_v1.go` `validateArchiveV1()`

**Step 5: Run tests**

Run: `cd /home/sasha/eng/repos/dodder/go && go test -v -tags test,debug ./src/echo/inventory_archive/... ./src/india/blob_stores/...`
Expected: All pass.

**Step 6: Commit**

```
git add -A && git commit -m "feat: add encryption support to V1 archive data writer/reader"
```

---

### Task 8: Wire encryption through pack and read paths

**Files:**
- Modify: `go/src/india/blob_stores/pack_v0.go`
- Modify: `go/src/india/blob_stores/pack_v1.go`
- Modify: `go/src/india/blob_stores/store_inventory_archive.go`
- Modify: `go/src/india/blob_stores/store_inventory_archive_v1.go`

**Step 1: Add helper to resolve IOWrapper from config**

Both V0 and V1 stores need to resolve the config's `GetBlobEncryption()` markl
ID into an `interfaces.IOWrapper`. Add a helper or inline it.

The config's `GetBlobEncryption()` returns a `domain_interfaces.MarklId`. If
non-nil and non-null, call `GetIOWrapper()` on it to get the `IOWrapper`.

In `pack_v0.go` `packChunkArchive()`, replace the `, nil` with the real wrapper:
```go
	encryption := store.config.GetBlobEncryption()
	var encryptionWrapper interfaces.IOWrapper
	if encryption != nil && !encryption.IsNull() {
		if encryptionWrapper, err = encryption.GetIOWrapper(); err != nil {
			err = errors.Wrap(err)
			return dataPath, 0, err
		}
	}

	dataWriter, err := inventory_archive.NewDataWriter(
		tmpFile,
		hashFormatId,
		ct,
		encryptionWrapper,
	)
```

Same pattern for:
- `pack_v0.go` `validateArchive()` → pass `encryptionWrapper` to `NewDataReader`
- `pack_v1.go` `packChunkArchiveV1()` → pass to `NewDataWriterV1`
- `pack_v1.go` `validateArchiveV1()` → pass to `NewDataReaderV1`
- `store_inventory_archive.go` `MakeBlobReader()` → pass to `NewDataReader`
- `store_inventory_archive.go` index rebuild → pass to `NewDataReader`
- `store_inventory_archive_v1.go` `MakeBlobReader()` → pass to `NewDataReaderV1`
- `store_inventory_archive_v1.go` index rebuild → pass to `NewDataReaderV1`

For the `validateArchive` and `MakeBlobReader` functions, the encryption wrapper
needs to be accessible. Either resolve it once and store on the struct, or
resolve it each time. Storing it during init is cleaner — add an
`encryptionWrapper interfaces.IOWrapper` field to `inventoryArchiveV0` and
`inventoryArchiveV1`, resolved during `makeInventoryArchiveV0`/`V1`.

**Step 2: Verify compilation**

Run: `cd /home/sasha/eng/repos/dodder/go && go build ./...`
Expected: Clean build.

**Step 3: Run tests**

Run: `cd /home/sasha/eng/repos/dodder/go && go test -v -tags test,debug ./src/india/blob_stores/...`
Expected: All pass (existing tests use no encryption, so IOWrapper is nil).

**Step 4: Commit**

```
git add -A && git commit -m "feat: wire encryption IOWrapper through pack and read paths"
```

---

### Task 9: Full test suite verification

**Step 1: Run unit tests**

Run: `cd /home/sasha/eng/repos/dodder/go && go test -v -tags test,debug ./...`
Expected: All pass.

**Step 2: Run BATS integration tests**

Run: `cd /home/sasha/eng/repos/dodder && /home/sasha/eng/result/bin/just test-bats`
Expected: All pass.

**Step 3: Format check**

Run: `cd /home/sasha/eng/repos/dodder && /home/sasha/eng/result/bin/just codemod-go-fmt`

If anything changed, commit it.

---

## Summary of Commits

1. `refactor: rename CompressedSize->StoredSize, UncompressedSize->LogicalSize`
2. `refactor: rename local vars and comments to stored_size/logical_size`
3. `refactor: rename archiveEntry.CompressedSize->StoredSize`
4. `feat: add FlagHasEncryption constants for archive format`
5. `feat: add encryption support to V0 archive data writer/reader`
6. `feat: add encryption support to V1 archive data writer/reader`
7. `feat: wire encryption IOWrapper through pack and read paths`
