# Delta Compression for Inventory Archives Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add delta compression to inventory archive packfiles (v1 format), with pluggable delta algorithms and base selection strategies.

**Architecture:** V1 archive format adds per-entry encoding metadata and delta entry types. V0 and v1 are horizontal implementations (separate structs, shared interfaces). A `DeltaAlgorithm` interface wraps xdelta/VCDIFF, a `BaseSelector` interface uses a dialog pattern for base selection. Config evolves from `TomlInventoryArchiveV0` to `TomlInventoryArchiveV1` with delta settings.

**Tech Stack:** Go, binary encoding (`encoding/binary`), xdelta/VCDIFF Go library, existing compression (`charlie/compression_type`), existing hash (`echo/markl`), TOML config via triple-hyphen-io.

**Reference:** See `docs/plans/2026-02-21-delta-compression-design.md` for the full design rationale.

---

### Task 1: Add disambiguation comments to local hash-bucketed config files

**Files:**
- Modify: `go/src/golf/blob_store_configs/toml_v0.go`
- Modify: `go/src/golf/blob_store_configs/toml_v1.go`
- Modify: `go/src/golf/blob_store_configs/toml_v2.go`

**Step 1: Add comments to each file**

Add a comment block above the struct definition in each file:

`toml_v0.go` — above the `TomlV0` struct:
```go
// TomlV0 is the V0 configuration for the local hash-bucketed blob store.
// TODO rename to TomlLocalHashBucketedV0 for disambiguation from other
// blob store config types (e.g., TomlInventoryArchiveV0).
```

`toml_v1.go` — above the `TomlV1` struct:
```go
// TomlV1 is the V1 configuration for the local hash-bucketed blob store.
// TODO rename to TomlLocalHashBucketedV1 for disambiguation from other
// blob store config types (e.g., TomlInventoryArchiveV1).
```

`toml_v2.go` — above the `TomlV2` struct:
```go
// TomlV2 is the V2 configuration for the local hash-bucketed blob store.
// TODO rename to TomlLocalHashBucketedV2 for disambiguation from other
// blob store config types (e.g., TomlInventoryArchiveV1).
```

**Step 2: Verify it compiles**

Run: `cd go && go build ./src/golf/blob_store_configs/`
Expected: success, no errors.

**Step 3: Commit**

```
chore(golf/blob_store_configs): add disambiguation comments to local hash-bucketed configs
```

---

### Task 2: Add v1 type constants and entry types to `echo/inventory_archive`

**Files:**
- Modify: `go/src/echo/inventory_archive/types.go`

**Step 1: Add v1 constants**

Add after the existing v0 constants block (after line 30 in `types.go`):

```go
const (
	DataFileVersionV1  uint16 = 1
	IndexFileVersionV1 uint16 = 1
	CacheFileVersionV1 uint16 = 1

	DataFileExtensionV1  = ".inventory_archive-v1"
	IndexFileExtensionV1 = ".inventory_archive_index-v1"
	CacheFileNameV1      = "index_cache-v1"

	EntryTypeFull  byte = 0x00
	EntryTypeDelta byte = 0x01

	FlagHasDeltas         uint16 = 1 << 0
	FlagReservedCrossArch uint16 = 1 << 1
)
```

**Step 2: Add v1 entry struct types**

Add after the existing `CacheEntry` struct (after line 51):

```go
type DataEntryV1 struct {
	Hash             []byte
	EntryType        byte
	Encoding         byte
	UncompressedSize uint64
	CompressedSize   uint64
	Data             []byte
	Offset           uint64
	// Delta-specific fields (only set when EntryType == EntryTypeDelta)
	DeltaAlgorithm byte
	BaseHash       []byte
}

type IndexEntryV1 struct {
	Hash           []byte
	PackOffset     uint64
	CompressedSize uint64
	EntryType      byte
	BaseOffset     uint64
}

type CacheEntryV1 struct {
	Hash            []byte
	ArchiveChecksum []byte
	Offset          uint64
	CompressedSize  uint64
	EntryType       byte
	BaseOffset      uint64
}
```

**Step 3: Verify it compiles**

Run: `cd go && go build ./src/echo/inventory_archive/`
Expected: success, no errors.

**Step 4: Commit**

```
feat(echo/inventory_archive): add v1 constants and entry types for delta compression
```

---

### Task 3: Add delta algorithm interface and registry

**Files:**
- Create: `go/src/echo/inventory_archive/delta_algorithm.go`

**Step 1: Write the failing test**

Create `go/src/echo/inventory_archive/delta_algorithm_test.go`:

```go
//go:build test && debug

package inventory_archive

import (
	"testing"
)

func TestDeltaAlgorithmRegistryLookup(t *testing.T) {
	alg, err := DeltaAlgorithmForByte(DeltaAlgorithmByteXdelta)
	if err != nil {
		t.Fatalf("expected xdelta algorithm, got error: %v", err)
	}

	if alg.Id() != DeltaAlgorithmByteXdelta {
		t.Errorf("expected id %d, got %d", DeltaAlgorithmByteXdelta, alg.Id())
	}
}

func TestDeltaAlgorithmRegistryUnknown(t *testing.T) {
	_, err := DeltaAlgorithmForByte(0xFF)
	if err == nil {
		t.Fatal("expected error for unknown algorithm byte")
	}
}

func TestDeltaAlgorithmNameLookup(t *testing.T) {
	b, err := DeltaAlgorithmByteForName("xdelta")
	if err != nil {
		t.Fatalf("expected xdelta byte, got error: %v", err)
	}

	if b != DeltaAlgorithmByteXdelta {
		t.Errorf("expected byte %d, got %d", DeltaAlgorithmByteXdelta, b)
	}
}

func TestDeltaAlgorithmNameUnknown(t *testing.T) {
	_, err := DeltaAlgorithmByteForName("unknown")
	if err == nil {
		t.Fatal("expected error for unknown algorithm name")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd go && go test -v -tags test,debug ./src/echo/inventory_archive/ -run TestDeltaAlgorithm`
Expected: FAIL — types not defined.

**Step 3: Write the interface and registry**

Create `go/src/echo/inventory_archive/delta_algorithm.go`:

```go
package inventory_archive

import (
	"io"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

// DeltaAlgorithm computes and applies binary deltas between blobs.
type DeltaAlgorithm interface {
	// Id returns the byte identifier written to data file delta entries.
	Id() byte

	// Compute produces a delta that transforms base into target.
	// The delta is written to the delta writer. base is a BlobReader
	// because current compression/encryption does not support seeking;
	// when BlobReader gains full ReadAtSeeker support, delta algorithms
	// can use random access for better performance.
	Compute(
		base domain_interfaces.BlobReader,
		baseSize int64,
		target io.Reader,
		delta io.Writer,
	) error

	// Apply reconstructs the original blob from a base and a delta.
	Apply(
		base domain_interfaces.BlobReader,
		baseSize int64,
		delta io.Reader,
		target io.Writer,
	) error
}

const (
	DeltaAlgorithmByteXdelta byte = 0
)

var deltaAlgorithms = map[byte]DeltaAlgorithm{}

var deltaAlgorithmNames = map[string]byte{
	"xdelta": DeltaAlgorithmByteXdelta,
}

// RegisterDeltaAlgorithm adds a DeltaAlgorithm to the registry.
func RegisterDeltaAlgorithm(alg DeltaAlgorithm) {
	deltaAlgorithms[alg.Id()] = alg
}

func DeltaAlgorithmForByte(b byte) (DeltaAlgorithm, error) {
	alg, ok := deltaAlgorithms[b]
	if !ok {
		return nil, errors.Errorf("unsupported delta algorithm byte: %d", b)
	}

	return alg, nil
}

func DeltaAlgorithmByteForName(name string) (byte, error) {
	b, ok := deltaAlgorithmNames[name]
	if !ok {
		return 0, errors.Errorf("unsupported delta algorithm name: %q", name)
	}

	return b, nil
}
```

**Step 4: Run test to verify it passes**

Note: The `TestDeltaAlgorithmRegistryLookup` test will fail until the xdelta
implementation is registered (Task 4). For now, verify the name lookup and
unknown-byte tests pass, and the registry-lookup test fails with the expected
"unsupported delta algorithm byte" error (not a compile error).

Run: `cd go && go test -v -tags test,debug ./src/echo/inventory_archive/ -run TestDeltaAlgorithm`
Expected: `TestDeltaAlgorithmNameLookup` PASS, `TestDeltaAlgorithmNameUnknown` PASS, `TestDeltaAlgorithmRegistryUnknown` PASS, `TestDeltaAlgorithmRegistryLookup` FAIL (xdelta not registered yet).

**Step 5: Commit**

```
feat(echo/inventory_archive): add DeltaAlgorithm interface and registry
```

---

### Task 4: Implement xdelta delta algorithm

**Files:**
- Create: `go/src/echo/inventory_archive/delta_xdelta.go`
- Create: `go/src/echo/inventory_archive/delta_xdelta_test.go`

**Step 1: Add xdelta Go dependency**

Research and select a Go xdelta/VCDIFF library. Candidates include
`github.com/ianatha/go-xdelta`, `github.com/gabstv/go-bsdiff`, or another
VCDIFF implementation. Choose the one with the best API fit for the
`DeltaAlgorithm` interface (byte-slice or streaming compute/apply).

Run: `cd go && go get <chosen-library>`

**Step 2: Write the failing test**

Create `go/src/echo/inventory_archive/delta_xdelta_test.go`:

```go
//go:build test && debug

package inventory_archive

import (
	"bytes"
	"io"
	"testing"

	"code.linenisgreat.com/dodder/go/src/bravo/markl_io"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
)

func makeMockBlobReader(data []byte) domain_interfaces.BlobReader {
	// Use markl_io.MakeNopReadCloser or MakeReadCloser with a
	// sha256 hash and bytes.NewReader to satisfy BlobReader interface.
	// Adapt to whatever test helper is available.
	hash, _ := markl.FormatHashSha256.Get()
	return markl_io.MakeReadCloser(hash, bytes.NewReader(data))
}

func TestXdeltaRoundTrip(t *testing.T) {
	base := []byte("the quick brown fox jumps over the lazy dog")
	target := []byte("the quick brown cat jumps over the lazy dog")

	alg := &Xdelta{}

	var deltaBuf bytes.Buffer

	baseReader := makeMockBlobReader(base)
	err := alg.Compute(
		baseReader,
		int64(len(base)),
		bytes.NewReader(target),
		&deltaBuf,
	)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	if deltaBuf.Len() == 0 {
		t.Fatal("expected non-empty delta")
	}

	if deltaBuf.Len() >= len(target) {
		t.Logf(
			"warning: delta (%d) not smaller than target (%d)",
			deltaBuf.Len(),
			len(target),
		)
	}

	var reconstructed bytes.Buffer

	baseReader2 := makeMockBlobReader(base)
	err = alg.Apply(
		baseReader2,
		int64(len(base)),
		&deltaBuf,
		&reconstructed,
	)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if !bytes.Equal(reconstructed.Bytes(), target) {
		t.Errorf(
			"reconstructed data mismatch: got %q, want %q",
			reconstructed.Bytes(),
			target,
		)
	}
}

func TestXdeltaIdenticalBlobs(t *testing.T) {
	data := []byte("identical content")

	alg := &Xdelta{}

	var deltaBuf bytes.Buffer

	baseReader := makeMockBlobReader(data)
	err := alg.Compute(
		baseReader,
		int64(len(data)),
		bytes.NewReader(data),
		&deltaBuf,
	)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	var reconstructed bytes.Buffer

	baseReader2 := makeMockBlobReader(data)
	err = alg.Apply(
		baseReader2,
		int64(len(data)),
		&deltaBuf,
		&reconstructed,
	)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if !bytes.Equal(reconstructed.Bytes(), data) {
		t.Errorf("reconstructed data mismatch for identical blobs")
	}
}

func TestXdeltaId(t *testing.T) {
	alg := &Xdelta{}
	if alg.Id() != DeltaAlgorithmByteXdelta {
		t.Errorf("expected id %d, got %d", DeltaAlgorithmByteXdelta, alg.Id())
	}
}

func TestXdeltaRegistered(t *testing.T) {
	alg, err := DeltaAlgorithmForByte(DeltaAlgorithmByteXdelta)
	if err != nil {
		t.Fatalf("xdelta should be registered: %v", err)
	}

	if alg.Id() != DeltaAlgorithmByteXdelta {
		t.Errorf("expected id %d, got %d", DeltaAlgorithmByteXdelta, alg.Id())
	}
}
```

Note: the `makeMockBlobReader` helper may need adjustment based on the exact
`markl_io` API. Check `src/bravo/markl_io/reader.go` for the correct
constructor. The key requirement is that the returned object satisfies
`domain_interfaces.BlobReader` and reads from the given byte slice.

**Step 3: Run test to verify it fails**

Run: `cd go && go test -v -tags test,debug ./src/echo/inventory_archive/ -run TestXdelta`
Expected: FAIL — `Xdelta` type not defined.

**Step 4: Implement the xdelta algorithm**

Create `go/src/echo/inventory_archive/delta_xdelta.go`:

```go
package inventory_archive

import (
	"io"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	// Import chosen xdelta library
)

func init() {
	RegisterDeltaAlgorithm(&Xdelta{})
}

// Xdelta implements DeltaAlgorithm using VCDIFF binary delta encoding.
type Xdelta struct{}

var _ DeltaAlgorithm = &Xdelta{}

func (x *Xdelta) Id() byte {
	return DeltaAlgorithmByteXdelta
}

func (x *Xdelta) Compute(
	base domain_interfaces.BlobReader,
	baseSize int64,
	target io.Reader,
	delta io.Writer,
) error {
	// Read the full base into memory. This is necessary because xdelta
	// requires random access to the base for hash table construction.
	// When BlobReader gains full ReadAtSeeker support through
	// compression/encryption, this can be optimized.
	baseData, err := io.ReadAll(base)
	if err != nil {
		return errors.Wrap(err)
	}

	targetData, err := io.ReadAll(target)
	if err != nil {
		return errors.Wrap(err)
	}

	// Use the chosen xdelta library to compute the delta.
	// Adapt this to the actual library API.
	deltaData, err := xdeltaCompute(baseData, targetData)
	if err != nil {
		return errors.Wrap(err)
	}

	if _, err := delta.Write(deltaData); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (x *Xdelta) Apply(
	base domain_interfaces.BlobReader,
	baseSize int64,
	delta io.Reader,
	target io.Writer,
) error {
	baseData, err := io.ReadAll(base)
	if err != nil {
		return errors.Wrap(err)
	}

	deltaData, err := io.ReadAll(delta)
	if err != nil {
		return errors.Wrap(err)
	}

	// Use the chosen xdelta library to apply the delta.
	targetData, err := xdeltaApply(baseData, deltaData)
	if err != nil {
		return errors.Wrap(err)
	}

	if _, err := target.Write(targetData); err != nil {
		return errors.Wrap(err)
	}

	return nil
}
```

The `xdeltaCompute` and `xdeltaApply` functions are wrappers around the chosen
library. Replace them with the actual API calls.

**Step 5: Run test to verify it passes**

Run: `cd go && go test -v -tags test,debug ./src/echo/inventory_archive/ -run TestXdelta`
Expected: all PASS.

**Step 6: Commit**

```
feat(echo/inventory_archive): implement xdelta DeltaAlgorithm
```

---

### Task 5: Add base selection interfaces

**Files:**
- Create: `go/src/echo/inventory_archive/base_selector.go`

**Step 1: Write the interface file**

Create `go/src/echo/inventory_archive/base_selector.go`:

```go
package inventory_archive

import (
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
)

// BlobMetadata describes a blob candidate for delta packing.
type BlobMetadata struct {
	Id   domain_interfaces.MarklId
	Size uint64
}

// BlobSet provides indexed access to blob metadata without requiring all
// blobs in memory simultaneously.
type BlobSet interface {
	Len() int
	At(index int) BlobMetadata
}

// DeltaAssignments receives base selection results. The packer passes this
// to the strategy, which calls Assign for each blob that should be
// delta-encoded.
type DeltaAssignments interface {
	// Assign records that the blob at blobIndex should be delta-encoded
	// against the blob at baseIndex. Both indices refer to the BlobSet.
	// Not calling Assign for a given index means store it as a full entry.
	Assign(blobIndex, baseIndex int)

	// AssignError reports that the strategy encountered an error for the
	// blob at blobIndex. The packer decides how to handle these.
	AssignError(blobIndex int, err error)
}

// BaseSelector chooses which blobs become deltas and which become bases.
// It reads from blobs and writes results to assignments.
type BaseSelector interface {
	SelectBases(blobs BlobSet, assignments DeltaAssignments)
}
```

**Step 2: Verify it compiles**

Run: `cd go && go build ./src/echo/inventory_archive/`
Expected: success.

**Step 3: Commit**

```
feat(echo/inventory_archive): add BaseSelector, BlobSet, DeltaAssignments interfaces
```

---

### Task 6: Implement size-based base selection strategy

**Files:**
- Create: `go/src/echo/inventory_archive/base_selector_size.go`
- Create: `go/src/echo/inventory_archive/base_selector_size_test.go`

**Step 1: Write the failing test**

Create `go/src/echo/inventory_archive/base_selector_size_test.go`:

```go
//go:build test && debug

package inventory_archive

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

type testBlobSet struct {
	blobs []BlobMetadata
}

func (s *testBlobSet) Len() int                  { return len(s.blobs) }
func (s *testBlobSet) At(index int) BlobMetadata { return s.blobs[index] }

type testAssignments struct {
	assignments map[int]int
	errors      map[int]error
}

func newTestAssignments() *testAssignments {
	return &testAssignments{
		assignments: make(map[int]int),
		errors:      make(map[int]error),
	}
}

func (a *testAssignments) Assign(blobIndex, baseIndex int) {
	a.assignments[blobIndex] = baseIndex
}

func (a *testAssignments) AssignError(blobIndex int, err error) {
	a.errors[blobIndex] = err
}

func makeTestMarklId(i int) domain_interfaces.MarklId {
	// Create a minimal MarklId from a sha256 hash for testing.
	// Use the markl package's test helpers or create a mock.
	data := []byte(fmt.Sprintf("blob-%04d", i))
	h := sha256.Sum256(data)
	id, _ := markl.FormatHashSha256.GetMarklIdForString(
		fmt.Sprintf("%x", h[:]),
	)
	return id
}

func TestSizeBasedSelectorGroupsSimilarSizes(t *testing.T) {
	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Id: makeTestMarklId(0), Size: 1000},
			{Id: makeTestMarklId(1), Size: 1100},
			{Id: makeTestMarklId(2), Size: 1200},
			{Id: makeTestMarklId(3), Size: 5000},
			{Id: makeTestMarklId(4), Size: 5500},
		},
	}

	selector := &SizeBasedSelector{
		MinBlobSize: 100,
		MaxBlobSize: 10000,
		SizeRatio:   2.0,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	// Blobs 0, 1, 2 should be grouped (within 2x ratio)
	// Blobs 3, 4 should be grouped (within 2x ratio)
	// The largest in each group should be the base
	if len(assignments.assignments) == 0 {
		t.Fatal("expected some delta assignments")
	}

	// Verify no self-assignments
	for blobIdx, baseIdx := range assignments.assignments {
		if blobIdx == baseIdx {
			t.Errorf("blob %d assigned to itself", blobIdx)
		}
	}
}

func TestSizeBasedSelectorSkipsSmallBlobs(t *testing.T) {
	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Id: makeTestMarklId(0), Size: 50},
			{Id: makeTestMarklId(1), Size: 60},
		},
	}

	selector := &SizeBasedSelector{
		MinBlobSize: 100,
		MaxBlobSize: 10000,
		SizeRatio:   2.0,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	if len(assignments.assignments) != 0 {
		t.Errorf("expected no assignments for small blobs, got %d",
			len(assignments.assignments))
	}
}

func TestSizeBasedSelectorSkipsLargeBlobs(t *testing.T) {
	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Id: makeTestMarklId(0), Size: 20000000},
			{Id: makeTestMarklId(1), Size: 20000001},
		},
	}

	selector := &SizeBasedSelector{
		MinBlobSize: 100,
		MaxBlobSize: 10000000,
		SizeRatio:   2.0,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	if len(assignments.assignments) != 0 {
		t.Errorf("expected no assignments for large blobs, got %d",
			len(assignments.assignments))
	}
}

func TestSizeBasedSelectorSingleBlob(t *testing.T) {
	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Id: makeTestMarklId(0), Size: 1000},
		},
	}

	selector := &SizeBasedSelector{
		MinBlobSize: 100,
		MaxBlobSize: 10000,
		SizeRatio:   2.0,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	if len(assignments.assignments) != 0 {
		t.Errorf("expected no assignments for single blob, got %d",
			len(assignments.assignments))
	}
}

func TestSizeBasedSelectorEmptyBlobSet(t *testing.T) {
	blobs := &testBlobSet{blobs: nil}

	selector := &SizeBasedSelector{
		MinBlobSize: 100,
		MaxBlobSize: 10000,
		SizeRatio:   2.0,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	if len(assignments.assignments) != 0 {
		t.Errorf("expected no assignments for empty blob set, got %d",
			len(assignments.assignments))
	}
}
```

Note: `makeTestMarklId` may need adaptation based on the actual `markl` test
API. Check `src/echo/markl/` for test helpers. The key is producing a valid
`domain_interfaces.MarklId`.

**Step 2: Run test to verify it fails**

Run: `cd go && go test -v -tags test,debug ./src/echo/inventory_archive/ -run TestSizeBased`
Expected: FAIL — `SizeBasedSelector` not defined.

**Step 3: Implement the size-based selector**

Create `go/src/echo/inventory_archive/base_selector_size.go`:

```go
package inventory_archive

import (
	"sort"
)

// SizeBasedSelector groups blobs by similar size and assigns deltas within
// each group against the largest blob as the base.
//
// TODO: Content-type base selection strategy — madder queries dodder for
// blob type info (binary flag), groups text blobs separately from binary.
//
// TODO: Object-history base selection strategy — dodder provides
// related-object hash chains, packer deltas successive versions of the
// same object against each other.
type SizeBasedSelector struct {
	MinBlobSize uint64
	MaxBlobSize uint64
	SizeRatio   float64
}

var _ BaseSelector = &SizeBasedSelector{}

func (s *SizeBasedSelector) SelectBases(
	blobs BlobSet,
	assignments DeltaAssignments,
) {
	n := blobs.Len()
	if n < 2 {
		return
	}

	// Build index sorted by size
	type indexedBlob struct {
		originalIndex int
		size          uint64
	}

	sorted := make([]indexedBlob, 0, n)

	for i := range n {
		meta := blobs.At(i)

		if meta.Size < s.MinBlobSize || meta.Size > s.MaxBlobSize {
			continue
		}

		sorted = append(sorted, indexedBlob{
			originalIndex: i,
			size:          meta.Size,
		})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].size < sorted[j].size
	})

	if len(sorted) < 2 {
		return
	}

	// Walk sorted list, grouping blobs within size ratio
	groupStart := 0
	for i := 1; i <= len(sorted); i++ {
		inGroup := i < len(sorted) &&
			float64(sorted[i].size) <= float64(sorted[groupStart].size)*s.SizeRatio

		if inGroup {
			continue
		}

		// End of group: [groupStart, i)
		if i-groupStart >= 2 {
			// Largest blob in group is the base (last in sorted order)
			baseIdx := sorted[i-1].originalIndex

			for j := groupStart; j < i-1; j++ {
				assignments.Assign(sorted[j].originalIndex, baseIdx)
			}
		}

		groupStart = i
	}
}
```

**Step 4: Run test to verify it passes**

Run: `cd go && go test -v -tags test,debug ./src/echo/inventory_archive/ -run TestSizeBased`
Expected: all PASS.

**Step 5: Commit**

```
feat(echo/inventory_archive): implement size-based BaseSelector strategy
```

---

### Task 7: Register v1 type constants

**Files:**
- Modify: `go/src/echo/ids/types_builtin.go`

**Step 1: Add type constants**

Add after the existing `TypeTomlBlobStoreConfigInventoryArchiveV0` constant:

```go
TypeTomlBlobStoreConfigInventoryArchiveV1       = "!toml-blob_store_config-inventory_archive-v1"
TypeTomlBlobStoreConfigInventoryArchiveVCurrent  = TypeTomlBlobStoreConfigInventoryArchiveV1
```

**Step 2: Register in init()**

Add after the existing `TypeTomlBlobStoreConfigInventoryArchiveV0` registration:

```go
registerBuiltinTypeString(
	TypeTomlBlobStoreConfigInventoryArchiveV1,
	genres.Unknown,
	false,
)
```

**Step 3: Verify it compiles**

Run: `cd go && go build ./src/echo/ids/`
Expected: success.

**Step 4: Commit**

```
feat(echo/ids): register inventory archive v1 config type constant
```

---

### Task 8: Add delta config interface and `TomlInventoryArchiveV1`

**Files:**
- Modify: `go/src/golf/blob_store_configs/main.go`
- Create: `go/src/golf/blob_store_configs/toml_inventory_archive_v1.go`
- Modify: `go/src/golf/blob_store_configs/toml_inventory_archive_v0.go`

**Step 1: Add `DeltaConfigImmutable` and `ConfigInventoryArchiveDelta` interfaces**

In `go/src/golf/blob_store_configs/main.go`, add after the
`ConfigInventoryArchive` interface block:

```go
DeltaConfigImmutable interface {
	GetDeltaEnabled() bool
	GetDeltaAlgorithm() string
	GetDeltaMinBlobSize() uint64
	GetDeltaMaxBlobSize() uint64
	GetDeltaSizeRatio() float64
}

ConfigInventoryArchiveDelta interface {
	ConfigInventoryArchive
	DeltaConfigImmutable
}
```

**Step 2: Create `TomlInventoryArchiveV1`**

Create `go/src/golf/blob_store_configs/toml_inventory_archive_v1.go`:

```go
package blob_store_configs

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
)

type DeltaConfig struct {
	Enabled     bool    `toml:"enabled"`
	Algorithm   string  `toml:"algorithm"`
	MinBlobSize uint64  `toml:"min-blob-size"`
	MaxBlobSize uint64  `toml:"max-blob-size"`
	SizeRatio   float64 `toml:"size-ratio"`
}

type TomlInventoryArchiveV1 struct {
	HashTypeId       string                           `toml:"hash_type-id"`
	CompressionType  compression_type.CompressionType `toml:"compression-type"`
	LooseBlobStoreId blob_store_id.Id                 `toml:"loose-blob-store-id"`
	Encryption       markl.Id                         `toml:"encryption"`
	Delta            DeltaConfig                      `toml:"delta"`
}

var (
	_ ConfigInventoryArchiveDelta = TomlInventoryArchiveV1{}
	_ ConfigMutable               = &TomlInventoryArchiveV1{}
	_                             = registerToml[TomlInventoryArchiveV1](
		Coder.Blob,
		ids.TypeTomlBlobStoreConfigInventoryArchiveV1,
	)
)

func (TomlInventoryArchiveV1) GetBlobStoreType() string {
	return "local-inventory-archive"
}

func (config *TomlInventoryArchiveV1) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	config.CompressionType.SetFlagDefinitions(flagSet)

	flagSet.StringVar(
		&config.HashTypeId,
		"hash_type-id",
		markl.FormatIdHashBlake2b256,
		"hash type for archive checksums and blob hashes",
	)

	flagSet.Var(
		&config.LooseBlobStoreId,
		"loose-blob-store-id",
		"id of the loose blob store to read from and write to",
	)
}

func (config TomlInventoryArchiveV1) getBasePath() string {
	return ""
}

func (config TomlInventoryArchiveV1) SupportsMultiHash() bool {
	return false
}

func (config TomlInventoryArchiveV1) GetDefaultHashTypeId() string {
	return config.HashTypeId
}

func (config TomlInventoryArchiveV1) GetBlobCompression() interfaces.IOWrapper {
	return &config.CompressionType
}

func (config TomlInventoryArchiveV1) GetBlobEncryption() domain_interfaces.MarklId {
	return config.Encryption
}

func (config TomlInventoryArchiveV1) GetLooseBlobStoreId() blob_store_id.Id {
	return config.LooseBlobStoreId
}

func (config TomlInventoryArchiveV1) GetCompressionType() compression_type.CompressionType {
	return config.CompressionType
}

// DeltaConfigImmutable implementation

func (config TomlInventoryArchiveV1) GetDeltaEnabled() bool {
	return config.Delta.Enabled
}

func (config TomlInventoryArchiveV1) GetDeltaAlgorithm() string {
	return config.Delta.Algorithm
}

func (config TomlInventoryArchiveV1) GetDeltaMinBlobSize() uint64 {
	return config.Delta.MinBlobSize
}

func (config TomlInventoryArchiveV1) GetDeltaMaxBlobSize() uint64 {
	return config.Delta.MaxBlobSize
}

func (config TomlInventoryArchiveV1) GetDeltaSizeRatio() float64 {
	return config.Delta.SizeRatio
}
```

**Step 3: Add `Upgrade()` to `TomlInventoryArchiveV0`**

In `go/src/golf/blob_store_configs/toml_inventory_archive_v0.go`, add the
`ConfigUpgradeable` interface assertion and method:

```go
var _ ConfigUpgradeable = TomlInventoryArchiveV0{}
```

```go
func (config TomlInventoryArchiveV0) Upgrade() (Config, ids.TypeStruct) {
	upgraded := &TomlInventoryArchiveV1{
		HashTypeId:       config.HashTypeId,
		CompressionType:  config.CompressionType,
		LooseBlobStoreId: config.LooseBlobStoreId,
		Delta: DeltaConfig{
			Enabled:     false,
			Algorithm:   "xdelta",
			MinBlobSize: 256,
			MaxBlobSize: 10485760,
			SizeRatio:   2.0,
		},
	}

	upgraded.Encryption.ResetWithMarklId(config.Encryption)

	return upgraded, ids.GetOrPanic(
		ids.TypeTomlBlobStoreConfigInventoryArchiveV1,
	).TypeStruct
}
```

**Step 4: Verify it compiles**

Run: `cd go && go build ./src/golf/blob_store_configs/`
Expected: success.

**Step 5: Commit**

```
feat(golf/blob_store_configs): add TomlInventoryArchiveV1 with delta config
```

---

### Task 9: Implement v1 data writer

**Files:**
- Create: `go/src/echo/inventory_archive/data_writer_v1.go`
- Create: `go/src/echo/inventory_archive/data_writer_v1_test.go`

**Step 1: Write the failing test**

Create `go/src/echo/inventory_archive/data_writer_v1_test.go`:

```go
//go:build test && debug

package inventory_archive

import (
	"bytes"
	"crypto/sha256"
	"testing"

	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
)

func TestV1RoundTripFullEntriesOnly(t *testing.T) {
	var buf bytes.Buffer
	hashFormatId := "sha256"
	ct := compression_type.CompressionTypeNone

	writer, err := NewDataWriterV1(&buf, hashFormatId, ct)
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
			data: []byte("second entry"),
			hash: sha256Hash([]byte("second entry")),
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
		t.Fatalf("expected checksum length %d, got %d",
			sha256.Size, len(checksum))
	}

	if len(writtenEntries) != len(entries) {
		t.Fatalf("expected %d entries, got %d",
			len(entries), len(writtenEntries))
	}

	for _, we := range writtenEntries {
		if we.EntryType != EntryTypeFull {
			t.Errorf("expected entry type full, got %d", we.EntryType)
		}
	}

	// Read back
	reader, err := NewDataReaderV1(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("NewDataReaderV1: %v", err)
	}

	readEntries, err := reader.ReadAllEntries()
	if err != nil {
		t.Fatalf("ReadAllEntries: %v", err)
	}

	if len(readEntries) != len(entries) {
		t.Fatalf("expected %d entries, got %d",
			len(entries), len(readEntries))
	}

	for i, re := range readEntries {
		if !bytes.Equal(re.Data, entries[i].data) {
			t.Errorf("entry %d: data mismatch", i)
		}

		if re.EntryType != EntryTypeFull {
			t.Errorf("entry %d: expected type full, got %d", i, re.EntryType)
		}
	}
}

func TestV1RoundTripWithDelta(t *testing.T) {
	var buf bytes.Buffer
	hashFormatId := "sha256"
	ct := compression_type.CompressionTypeNone

	writer, err := NewDataWriterV1(&buf, hashFormatId, ct)
	if err != nil {
		t.Fatalf("NewDataWriterV1: %v", err)
	}

	baseData := []byte("the quick brown fox jumps over the lazy dog")
	baseHash := sha256Hash(baseData)
	targetData := []byte("the quick brown cat jumps over the lazy dog")
	targetHash := sha256Hash(targetData)

	// Write base as full entry
	if err := writer.WriteFullEntry(baseHash, baseData); err != nil {
		t.Fatalf("WriteFullEntry: %v", err)
	}

	// Compute a fake delta (in real usage this comes from DeltaAlgorithm)
	fakeDelta := []byte("fake-delta-payload")

	// Write target as delta entry
	if err := writer.WriteDeltaEntry(
		targetHash,
		DeltaAlgorithmByteXdelta,
		baseHash,
		uint64(len(targetData)),
		fakeDelta,
	); err != nil {
		t.Fatalf("WriteDeltaEntry: %v", err)
	}

	checksum, writtenEntries, err := writer.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	if len(checksum) == 0 {
		t.Fatal("expected non-empty checksum")
	}

	if len(writtenEntries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(writtenEntries))
	}

	if writtenEntries[0].EntryType != EntryTypeFull {
		t.Errorf("entry 0: expected full, got %d", writtenEntries[0].EntryType)
	}

	if writtenEntries[1].EntryType != EntryTypeDelta {
		t.Errorf("entry 1: expected delta, got %d", writtenEntries[1].EntryType)
	}

	if writtenEntries[1].DeltaAlgorithm != DeltaAlgorithmByteXdelta {
		t.Errorf("entry 1: wrong delta algorithm")
	}

	if !bytes.Equal(writtenEntries[1].BaseHash, baseHash) {
		t.Errorf("entry 1: wrong base hash")
	}

	// Read back
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

	// Full entry should have original data
	if !bytes.Equal(readEntries[0].Data, baseData) {
		t.Errorf("entry 0: data mismatch")
	}

	// Delta entry should have the delta payload (not reconstructed data)
	if !bytes.Equal(readEntries[1].Data, fakeDelta) {
		t.Errorf("entry 1: delta payload mismatch")
	}

	if readEntries[1].DeltaAlgorithm != DeltaAlgorithmByteXdelta {
		t.Errorf("entry 1: wrong delta algorithm on read")
	}

	if !bytes.Equal(readEntries[1].BaseHash, baseHash) {
		t.Errorf("entry 1: wrong base hash on read")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd go && go test -v -tags test,debug ./src/echo/inventory_archive/ -run TestV1`
Expected: FAIL — `NewDataWriterV1` not defined.

**Step 3: Implement v1 data writer and reader**

Create `go/src/echo/inventory_archive/data_writer_v1.go`. Follow the same
structure as `data_writer.go` but:

- Version is `DataFileVersionV1`
- Extension is `DataFileExtensionV1`
- Header includes `default_encoding` byte and `flags` (2 bytes, set
  `FlagHasDeltas` if any delta entries are written)
- `WriteFullEntry(hash, data)` writes `entry_type=0x00`, `encoding` byte,
  uncompressed/compressed size, compressed data
- `WriteDeltaEntry(hash, deltaAlgorithm, baseHash, uncompressedSize, deltaData)`
  writes `entry_type=0x01`, `encoding` byte, `delta_algorithm` byte, base hash,
  uncompressed size (of reconstructed blob), delta size, compressed delta data
- `Close()` returns `[]DataEntryV1` with populated `EntryType`, `DeltaAlgorithm`,
  `BaseHash` fields

Create `go/src/echo/inventory_archive/data_reader_v1.go` with corresponding
reader that parses both full and delta entries.

**Step 4: Run test to verify it passes**

Run: `cd go && go test -v -tags test,debug ./src/echo/inventory_archive/ -run TestV1`
Expected: all PASS.

**Step 5: Commit**

```
feat(echo/inventory_archive): implement v1 data writer and reader with delta entries
```

---

### Task 10: Implement v1 index writer and reader

**Files:**
- Create: `go/src/echo/inventory_archive/index_v1.go`
- Create: `go/src/echo/inventory_archive/index_v1_test.go`

**Step 1: Write the failing test**

Create `go/src/echo/inventory_archive/index_v1_test.go` following the pattern
in `index_test.go`, but using `IndexEntryV1` with `EntryType` and `BaseOffset`
fields. Test round-trip, lookup, fan-out, empty index, checksum validation, and
unsorted rejection.

**Step 2: Run test to verify it fails**

Run: `cd go && go test -v -tags test,debug ./src/echo/inventory_archive/ -run TestIndexV1`
Expected: FAIL.

**Step 3: Implement v1 index writer and reader**

Create `go/src/echo/inventory_archive/index_v1.go` following the `index_writer.go`
and `index_reader.go` patterns. Each index entry adds:
- `entry_type`: 1 byte
- `base_offset`: 8 bytes uint64 (0 for full entries)

Update `LookupHash` to also return `entry_type` and `base_offset`.

**Step 4: Run test to verify it passes**

Run: `cd go && go test -v -tags test,debug ./src/echo/inventory_archive/ -run TestIndexV1`
Expected: all PASS.

**Step 5: Commit**

```
feat(echo/inventory_archive): implement v1 index writer and reader
```

---

### Task 11: Implement v1 cache writer and reader

**Files:**
- Create: `go/src/echo/inventory_archive/cache_v1.go`
- Create: `go/src/echo/inventory_archive/cache_v1_test.go`

Follow the same pattern as Task 10 but for cache entries. `CacheEntryV1` adds
`EntryType` and `BaseOffset` fields.

**Step 1: Write failing test**
**Step 2: Run test to verify it fails**
**Step 3: Implement v1 cache writer and reader**
**Step 4: Run test to verify it passes**
**Step 5: Commit**

```
feat(echo/inventory_archive): implement v1 cache writer and reader
```

---

### Task 12: Extract v0 inventory archive store to horizontal implementation

**Files:**
- Rename: `go/src/india/blob_stores/store_inventory_archive.go` -> keep as shared/common code
- Create: `go/src/india/blob_stores/store_inventory_archive_v0.go`
- Rename: `go/src/india/blob_stores/pack.go` -> keep `PackableArchive` interface
- Create: `go/src/india/blob_stores/pack_v0.go`

This is a refactoring task. Extract the current `inventoryArchive` struct into
`inventoryArchiveV0`. Move the `Pack` method into `pack_v0.go`. Keep shared
helpers (archive entry types, common index loading) in the original files.

**Step 1: Create `store_inventory_archive_v0.go`**

Move the `inventoryArchive` struct and rename to `inventoryArchiveV0`. Move all
methods that are specific to v0 (using v0 constants, v0 file extensions).

**Step 2: Create `pack_v0.go`**

Move the `Pack` method from `pack.go` into `pack_v0.go` as a method on
`inventoryArchiveV0`. Keep the `PackableArchive` interface and `PackOptions` in
`pack.go`.

**Step 3: Update `main.go` factory**

Update `MakeBlobStore` to construct `inventoryArchiveV0` when the config is
`ConfigInventoryArchive` (not `ConfigInventoryArchiveDelta`).

**Step 4: Run all tests**

Run: `cd go && go test -v -tags test,debug ./src/india/blob_stores/`
Expected: all existing tests pass.

Run: `just test-go-unit` to verify nothing else broke.

**Step 5: Commit**

```
refactor(india/blob_stores): extract v0 inventory archive as horizontal implementation
```

---

### Task 13: Implement v1 inventory archive store with delta packing

**Files:**
- Create: `go/src/india/blob_stores/store_inventory_archive_v1.go`
- Create: `go/src/india/blob_stores/pack_v1.go`
- Create: `go/src/india/blob_stores/pack_v1_test.go`
- Modify: `go/src/india/blob_stores/main.go`

**Step 1: Create `store_inventory_archive_v1.go`**

New `inventoryArchiveV1` struct implementing `domain_interfaces.BlobStore`. Uses
v1 file extensions and v1 reader for archive access. `MakeBlobReader` handles
delta reconstruction: if entry type is delta, reads the base entry, decompresses
it, reads the delta, decompresses it, looks up the delta algorithm, and calls
`Apply` to reconstruct.

**Step 2: Create `pack_v1.go`**

The `Pack` method on `inventoryArchiveV1`:

1. Collects loose blobs into a `BlobSet`
2. Creates `SizeBasedSelector` from config
3. Calls `SelectBases(blobSet, assignments)`
4. Writes full entries first via `DataWriterV1.WriteFullEntry`
5. For each delta assignment:
   - Opens base and target via `BlobReaderFactory`
   - Calls `DeltaAlgorithm.Compute(baseReader, baseSize, targetReader, deltaBuf)`
   - Compares compressed delta size vs compressed full size
   - If delta is smaller, writes via `WriteDeltaEntry`; otherwise writes as full
6. Builds v1 index and cache
7. Hard fails on any I/O error

```go
// TODO: Collect all blob failures during packing, present summary
// to user with interactive choices (retry individual, skip to full
// entry, abort). For now, hard fail on first error.
```

**Step 3: Update `main.go` factory**

Add a case for `ConfigInventoryArchiveDelta` in `MakeBlobStore`:

```go
case blob_store_configs.ConfigInventoryArchiveDelta:
	// V1 inventory archive with delta support
	return makeInventoryArchiveV1(
		envDir,
		configNamed.Path.GetBase(),
		config,
		looseBlobStore,
	)
```

This case must come before the `ConfigInventoryArchive` case in the type switch
because `ConfigInventoryArchiveDelta` embeds `ConfigInventoryArchive`.

**Step 4: Write integration test**

Create `go/src/india/blob_stores/pack_v1_test.go` that:
- Creates a loose blob store with several similar blobs
- Creates a v1 inventory archive store with delta enabled
- Calls `Pack`
- Verifies all blobs are readable via `MakeBlobReader`
- Verifies at least one entry was stored as a delta (archive is smaller than
  sum of original blobs)

**Step 5: Run tests**

Run: `cd go && go test -v -tags test,debug ./src/india/blob_stores/`
Expected: all PASS.

**Step 6: Commit**

```
feat(india/blob_stores): implement v1 inventory archive with delta packing
```

---

### Task 14: End-to-end integration test

**Files:**
- Modify or create BATS test in `zz-tests_bats/`

**Step 1: Build debug binary**

Run: `just build`

**Step 2: Write BATS test**

Create a BATS test that:
- Initializes a dodder repo with an inventory archive blob store (v1 config)
- Creates several zettels with similar content
- Runs `madder pack`
- Verifies all zettels are still readable
- Verifies archive files exist with `.inventory_archive-v1` extension

**Step 3: Run BATS test**

Run: `just test-bats-targets <test-file>.bats`
Expected: PASS.

**Step 4: Commit**

```
test: add BATS integration test for delta-compressed inventory archives
```

---

### Task 15: Run full test suite and format

**Step 1: Format all new/modified files**

Run: `just codemod-go-fmt`

**Step 2: Run full checks**

Run: `just check`
Expected: no vulnerabilities, vet passes, repool analyzer passes.

**Step 3: Run full test suite**

Run: `just test`
Expected: all unit tests and BATS integration tests pass.

**Step 4: Fix any failures, then commit**

```
chore: format and fix any issues from full test suite
```
