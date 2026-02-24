# Delta Similarity Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add content-similarity-based delta base selection to inventory archive packing using MinHash over CDC chunk hashes with LSH banding.

**Architecture:** A `SignatureComputer` interface produces fixed-length MinHash signatures from blob content via Gear-hash CDC chunking. An `LSHBandingSelector` implements `BaseSelector` using those signatures to find similar blob pairs in sublinear time. The packer computes signatures before calling the selector. Config pairs computer + selector.

**Tech Stack:** Go, no external dependencies (Gear hash table + MinHash + FNV hashing are all implementable in ~200 lines). Build tags: `test && debug` for tests.

---

### Task 1: Gear Hash CDC Chunking Primitives

**Files:**
- Create: `go/src/echo/inventory_archive/gear_hash.go`
- Test: `go/src/echo/inventory_archive/gear_hash_test.go`

**Step 1: Write the failing tests**

Create `gear_hash_test.go` with build tag `//go:build test && debug`:

```go
//go:build test && debug

package inventory_archive

import (
	"bytes"
	"testing"
)

func TestGearCDCProducesChunks(t *testing.T) {
	// 1KB of deterministic data should produce multiple chunks
	// with avg chunk size 64
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i * 37)
	}

	chunks := GearCDCChunks(data, 16, 256, 64)

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}

	// Verify chunks reconstruct original
	var reconstructed []byte
	for _, c := range chunks {
		reconstructed = append(reconstructed, c...)
	}

	if !bytes.Equal(data, reconstructed) {
		t.Fatal("chunks do not reconstruct original data")
	}
}

func TestGearCDCRespectsMinChunkSize(t *testing.T) {
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i)
	}

	minSize := 32
	chunks := GearCDCChunks(data, minSize, 256, 64)

	for i, c := range chunks {
		// Last chunk may be smaller than min
		if i < len(chunks)-1 && len(c) < minSize {
			t.Errorf("chunk %d has size %d < min %d", i, len(c), minSize)
		}
	}
}

func TestGearCDCRespectsMaxChunkSize(t *testing.T) {
	data := make([]byte, 4096)
	for i := range data {
		data[i] = byte(i)
	}

	maxSize := 128
	chunks := GearCDCChunks(data, 16, maxSize, 64)

	for i, c := range chunks {
		if len(c) > maxSize {
			t.Errorf("chunk %d has size %d > max %d", i, len(c), maxSize)
		}
	}
}

func TestGearCDCInsertionOnlyAffectsNearbyChunks(t *testing.T) {
	// Create original data
	original := make([]byte, 2048)
	for i := range original {
		original[i] = byte(i * 7)
	}

	// Insert 10 bytes in the middle
	insertion := make([]byte, len(original)+10)
	copy(insertion[:1024], original[:1024])
	copy(insertion[1024:1034], []byte("INSERTDATA"))
	copy(insertion[1034:], original[1024:])

	origChunks := GearCDCChunks(original, 16, 256, 64)
	insChunks := GearCDCChunks(insertion, 16, 256, 64)

	// Count matching chunks (by content)
	origSet := make(map[string]bool)
	for _, c := range origChunks {
		origSet[string(c)] = true
	}

	matching := 0
	for _, c := range insChunks {
		if origSet[string(c)] {
			matching++
		}
	}

	// Most chunks should survive the insertion (CDC property)
	matchRatio := float64(matching) / float64(len(origChunks))
	if matchRatio < 0.5 {
		t.Errorf(
			"insertion disrupted too many chunks: %d/%d matching (%.1f%%)",
			matching, len(origChunks), matchRatio*100,
		)
	}
}

func TestGearCDCEmptyInput(t *testing.T) {
	chunks := GearCDCChunks(nil, 16, 256, 64)
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty input, got %d", len(chunks))
	}
}

func TestGearCDCSmallInput(t *testing.T) {
	data := []byte("hello")
	chunks := GearCDCChunks(data, 16, 256, 64)

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for small input, got %d", len(chunks))
	}

	if !bytes.Equal(chunks[0], data) {
		t.Fatal("single chunk should equal input")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd go && go test -v -tags test,debug -run TestGearCDC ./src/echo/inventory_archive/`
Expected: compilation error — `GearCDCChunks` undefined

**Step 3: Implement Gear hash CDC**

Create `gear_hash.go`:

```go
package inventory_archive

// gearTable maps each byte value to a random 64-bit constant used by the
// Gear rolling hash. Generated deterministically so chunk boundaries are
// reproducible across runs.
var gearTable [256]uint64

func init() {
	// LCG seeded deterministically to produce the gear table.
	// Constants from Numerical Recipes.
	var state uint64 = 0x5F3759DF
	for i := range gearTable {
		state = state*6364136223846793005 + 1442695040888963407
		gearTable[i] = state
	}
}

// GearCDCChunks splits data into variable-length chunks using the Gear
// rolling hash (FastCDC-style). Chunk boundaries occur where the hash
// matches a mask derived from avgChunkSize. minChunkSize bytes are
// skipped after each boundary before checking again. Chunks are capped
// at maxChunkSize.
//
// Returns slices into the original data (no copies).
func GearCDCChunks(
	data []byte,
	minChunkSize, maxChunkSize, avgChunkSize int,
) [][]byte {
	if len(data) == 0 {
		return nil
	}

	// mask selects chunk boundaries: fp & mask == 0 gives avg chunk
	// size of (mask+1). We want avg ~= avgChunkSize, so mask =
	// avgChunkSize - 1 rounded down to a power of two minus one.
	mask := uint64(nextPowerOfTwo(avgChunkSize) - 1)

	var chunks [][]byte
	var fp uint64

	chunkStart := 0

	for i := 0; i < len(data); i++ {
		fp = (fp << 1) + gearTable[data[i]]

		chunkLen := i - chunkStart + 1

		if chunkLen < minChunkSize {
			continue
		}

		if chunkLen >= maxChunkSize || (fp&mask) == 0 {
			chunks = append(chunks, data[chunkStart:i+1])
			chunkStart = i + 1
			fp = 0
		}
	}

	// Remaining bytes form the last chunk
	if chunkStart < len(data) {
		chunks = append(chunks, data[chunkStart:])
	}

	return chunks
}

func nextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}

	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	n++

	return n
}
```

**Step 4: Run tests to verify they pass**

Run: `cd go && go test -v -tags test,debug -run TestGearCDC ./src/echo/inventory_archive/`
Expected: all PASS

**Step 5: Commit**

```
feat: add Gear hash CDC chunking primitives

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

---

### Task 2: MinHash One-Permutation Hashing

**Files:**
- Create: `go/src/echo/inventory_archive/minhash.go`
- Test: `go/src/echo/inventory_archive/minhash_test.go`

**Step 1: Write the failing tests**

Create `minhash_test.go` with build tag `//go:build test && debug`:

```go
//go:build test && debug

package inventory_archive

import (
	"math"
	"testing"
)

func TestMinHashIdenticalSetsProduceIdenticalSignatures(t *testing.T) {
	set := []uint32{100, 200, 300, 400, 500}

	sig1 := MinHashSignature(set, 16)
	sig2 := MinHashSignature(set, 16)

	for i := range sig1 {
		if sig1[i] != sig2[i] {
			t.Fatalf("position %d: %d != %d", i, sig1[i], sig2[i])
		}
	}
}

func TestMinHashDisjointSetsProduceDifferentSignatures(t *testing.T) {
	setA := []uint32{1, 2, 3, 4, 5}
	setB := []uint32{1001, 1002, 1003, 1004, 1005}

	sigA := MinHashSignature(setA, 64)
	sigB := MinHashSignature(setB, 64)

	matches := 0
	for i := range sigA {
		if sigA[i] == sigB[i] {
			matches++
		}
	}

	// Disjoint sets: Jaccard = 0, so matches should be near 0
	if float64(matches)/float64(len(sigA)) > 0.2 {
		t.Errorf("disjoint sets have too many matches: %d/%d", matches, len(sigA))
	}
}

func TestMinHashEstimatesJaccardSimilarity(t *testing.T) {
	// Create two sets with 80% overlap
	shared := make([]uint32, 80)
	for i := range shared {
		shared[i] = uint32(i)
	}

	setA := make([]uint32, 100)
	copy(setA[:80], shared)
	for i := 80; i < 100; i++ {
		setA[i] = uint32(1000 + i)
	}

	setB := make([]uint32, 100)
	copy(setB[:80], shared)
	for i := 80; i < 100; i++ {
		setB[i] = uint32(2000 + i)
	}

	// True Jaccard: 80 / (100 + 100 - 80) = 80/120 = 0.667
	expectedJaccard := 80.0 / 120.0

	k := 256
	sigA := MinHashSignature(setA, k)
	sigB := MinHashSignature(setB, k)

	estimated := MinHashJaccard(sigA, sigB)

	// With k=256, std error ~= sqrt(J*(1-J)/k) ~= 0.029
	// Allow 3 standard deviations
	tolerance := 3 * math.Sqrt(expectedJaccard*(1-expectedJaccard)/float64(k))

	if math.Abs(estimated-expectedJaccard) > tolerance {
		t.Errorf(
			"estimated Jaccard %.3f too far from expected %.3f (tolerance %.3f)",
			estimated, expectedJaccard, tolerance,
		)
	}
}

func TestMinHashSignatureLenMatchesK(t *testing.T) {
	set := []uint32{1, 2, 3}
	sig := MinHashSignature(set, 32)

	if len(sig) != 32 {
		t.Errorf("expected signature length 32, got %d", len(sig))
	}
}

func TestMinHashEmptySet(t *testing.T) {
	sig := MinHashSignature(nil, 16)

	if len(sig) != 16 {
		t.Errorf("expected signature length 16, got %d", len(sig))
	}

	// All positions should be MaxUint32 (no minimums found)
	for i, v := range sig {
		if v != math.MaxUint32 {
			t.Errorf("position %d: expected MaxUint32, got %d", i, v)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd go && go test -v -tags test,debug -run TestMinHash ./src/echo/inventory_archive/`
Expected: compilation error — `MinHashSignature` undefined

**Step 3: Implement MinHash**

Create `minhash.go`:

```go
package inventory_archive

import (
	"math"
)

// MinHashSignature computes a MinHash signature of length k over the
// given set of uint32 feature values. Uses one-permutation hashing:
// a single hash function partitions the hash space into k bins, and
// each bin tracks its minimum.
//
// The signature can be compared with MinHashJaccard to estimate the
// Jaccard similarity of the original sets.
func MinHashSignature(features []uint32, k int) []uint32 {
	sig := make([]uint32, k)
	for i := range sig {
		sig[i] = math.MaxUint32
	}

	for _, f := range features {
		h := minhashHash(f)
		bin := int(h % uint32(k))
		val := h / uint32(k)

		if val < sig[bin] {
			sig[bin] = val
		}
	}

	return sig
}

// MinHashJaccard estimates the Jaccard similarity between two sets
// from their MinHash signatures. Returns the fraction of positions
// where both signatures have the same non-empty value.
func MinHashJaccard(sigA, sigB []uint32) float64 {
	if len(sigA) != len(sigB) {
		return 0
	}

	matches := 0
	nonEmpty := 0

	for i := range sigA {
		// Skip bins where either signature had no elements
		if sigA[i] == math.MaxUint32 || sigB[i] == math.MaxUint32 {
			continue
		}

		nonEmpty++

		if sigA[i] == sigB[i] {
			matches++
		}
	}

	if nonEmpty == 0 {
		return 0
	}

	return float64(matches) / float64(nonEmpty)
}

// minhashHash is a fast 32-bit mixing function (murmur3 finalizer).
func minhashHash(x uint32) uint32 {
	x ^= x >> 16
	x *= 0x85ebca6b
	x ^= x >> 13
	x *= 0xc2b2ae35
	x ^= x >> 16
	return x
}
```

**Step 4: Run tests to verify they pass**

Run: `cd go && go test -v -tags test,debug -run TestMinHash ./src/echo/inventory_archive/`
Expected: all PASS

**Step 5: Commit**

```
feat: add MinHash one-permutation hashing

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

---

### Task 3: SignatureComputer Interface + GearCDCMinHash Implementation

**Files:**
- Create: `go/src/echo/inventory_archive/signature_computer.go`
- Create: `go/src/echo/inventory_archive/signature_gear_cdc_minhash.go`
- Test: `go/src/echo/inventory_archive/signature_gear_cdc_minhash_test.go`
- Modify: `go/src/echo/inventory_archive/base_selector.go` (add Signature field to BlobMetadata)

**Step 1: Write the failing tests**

Create `signature_gear_cdc_minhash_test.go` with build tag `//go:build test && debug`:

```go
//go:build test && debug

package inventory_archive

import (
	"bytes"
	"math"
	"testing"
)

func TestGearCDCMinHashComputerSignatureLen(t *testing.T) {
	computer := &GearCDCMinHashComputer{
		AvgChunkSize: 64,
		MinChunkSize: 16,
		MaxChunkSize: 256,
		K:            64,
	}

	if computer.SignatureLen() != 64 {
		t.Errorf("expected 64, got %d", computer.SignatureLen())
	}
}

func TestGearCDCMinHashComputerSimilarBlobs(t *testing.T) {
	computer := &GearCDCMinHashComputer{
		AvgChunkSize: 64,
		MinChunkSize: 16,
		MaxChunkSize: 256,
		K:            128,
	}

	original := make([]byte, 2048)
	for i := range original {
		original[i] = byte(i * 7)
	}

	// Small edit: change 5% of content
	edited := make([]byte, len(original))
	copy(edited, original)
	for i := 1000; i < 1100; i++ {
		edited[i] = byte(i * 13)
	}

	sigA, err := computer.ComputeSignature(bytes.NewReader(original))
	if err != nil {
		t.Fatal(err)
	}

	sigB, err := computer.ComputeSignature(bytes.NewReader(edited))
	if err != nil {
		t.Fatal(err)
	}

	similarity := MinHashJaccard(sigA, sigB)

	// With 5% edit and CDC, expect >0.5 similarity
	if similarity < 0.5 {
		t.Errorf("similar blobs have low similarity: %.3f", similarity)
	}
}

func TestGearCDCMinHashComputerDissimilarBlobs(t *testing.T) {
	computer := &GearCDCMinHashComputer{
		AvgChunkSize: 64,
		MinChunkSize: 16,
		MaxChunkSize: 256,
		K:            128,
	}

	blobA := make([]byte, 1024)
	for i := range blobA {
		blobA[i] = byte(i * 3)
	}

	blobB := make([]byte, 1024)
	for i := range blobB {
		blobB[i] = byte(i*17 + 128)
	}

	sigA, err := computer.ComputeSignature(bytes.NewReader(blobA))
	if err != nil {
		t.Fatal(err)
	}

	sigB, err := computer.ComputeSignature(bytes.NewReader(blobB))
	if err != nil {
		t.Fatal(err)
	}

	similarity := MinHashJaccard(sigA, sigB)

	if similarity > 0.3 {
		t.Errorf("dissimilar blobs have high similarity: %.3f", similarity)
	}
}

func TestGearCDCMinHashComputerIdenticalBlobs(t *testing.T) {
	computer := &GearCDCMinHashComputer{
		AvgChunkSize: 64,
		MinChunkSize: 16,
		MaxChunkSize: 256,
		K:            64,
	}

	data := []byte("the quick brown fox jumps over the lazy dog, repeatedly")

	sigA, err := computer.ComputeSignature(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	sigB, err := computer.ComputeSignature(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	similarity := MinHashJaccard(sigA, sigB)

	if similarity != 1.0 {
		t.Errorf("identical blobs should have similarity 1.0, got %.3f", similarity)
	}
}

func TestGearCDCMinHashComputerEmptyBlob(t *testing.T) {
	computer := &GearCDCMinHashComputer{
		AvgChunkSize: 64,
		MinChunkSize: 16,
		MaxChunkSize: 256,
		K:            16,
	}

	sig, err := computer.ComputeSignature(bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}

	if len(sig) != 16 {
		t.Errorf("expected len 16, got %d", len(sig))
	}

	for i, v := range sig {
		if v != math.MaxUint32 {
			t.Errorf("position %d: expected MaxUint32 for empty blob, got %d", i, v)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd go && go test -v -tags test,debug -run TestGearCDCMinHashComputer ./src/echo/inventory_archive/`
Expected: compilation error — types undefined

**Step 3: Add Signature field to BlobMetadata and create interfaces**

Modify `go/src/echo/inventory_archive/base_selector.go` — add `Signature []uint32` field to `BlobMetadata`:

```go
type BlobMetadata struct {
	Id        domain_interfaces.MarklId
	Size      uint64
	Signature []uint32
}
```

Create `go/src/echo/inventory_archive/signature_computer.go`:

```go
package inventory_archive

import "io"

// SignatureComputer produces a fixed-length similarity signature from
// blob content. Signatures from the same computer are comparable:
// the fraction of matching positions estimates content similarity.
type SignatureComputer interface {
	SignatureLen() int
	ComputeSignature(content io.Reader) ([]uint32, error)
}
```

Create `go/src/echo/inventory_archive/signature_gear_cdc_minhash.go`:

```go
package inventory_archive

import (
	"hash/fnv"
	"io"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

// GearCDCMinHashComputer splits blob content into variable-length
// chunks using Gear hash CDC, hashes each chunk with FNV-1a, and
// computes a MinHash signature over the chunk hash set.
type GearCDCMinHashComputer struct {
	AvgChunkSize int
	MinChunkSize int
	MaxChunkSize int
	K            int
}

var _ SignatureComputer = &GearCDCMinHashComputer{}

func (c *GearCDCMinHashComputer) SignatureLen() int {
	return c.K
}

func (c *GearCDCMinHashComputer) ComputeSignature(
	content io.Reader,
) ([]uint32, error) {
	data, err := io.ReadAll(content)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	chunks := GearCDCChunks(data, c.MinChunkSize, c.MaxChunkSize, c.AvgChunkSize)

	features := make([]uint32, len(chunks))
	h := fnv.New32a()

	for i, chunk := range chunks {
		h.Reset()
		h.Write(chunk)
		features[i] = h.Sum32()
	}

	return MinHashSignature(features, c.K), nil
}
```

**Step 4: Run tests to verify they pass**

Run: `cd go && go test -v -tags test,debug -run TestGearCDCMinHashComputer ./src/echo/inventory_archive/`
Expected: all PASS

**Step 5: Verify existing tests still pass**

Run: `cd go && go test -v -tags test,debug -run TestSizeBased ./src/echo/inventory_archive/`
Expected: all PASS (Signature field is zero-value nil, SizeBasedSelector ignores it)

**Step 6: Commit**

```
feat: add SignatureComputer interface and GearCDCMinHash implementation

Extend BlobMetadata with optional Signature field. Add SignatureComputer
interface and GearCDCMinHashComputer that uses Gear hash CDC chunking
with MinHash one-permutation hashing.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

---

### Task 4: LSH Banding Selector

**Files:**
- Create: `go/src/echo/inventory_archive/base_selector_lsh.go`
- Test: `go/src/echo/inventory_archive/base_selector_lsh_test.go`

**Step 1: Write the failing tests**

Create `base_selector_lsh_test.go` with build tag `//go:build test && debug`.

Use the `testBlobSet`, `testAssignments`, and `newTestAssignments` helpers
already defined in `base_selector_size_test.go` (same package, same build tag).

```go
//go:build test && debug

package inventory_archive

import (
	"bytes"
	"math"
	"testing"
)

func makeTestSignatures(
	t *testing.T,
	blobs [][]byte,
	computer *GearCDCMinHashComputer,
) [][]uint32 {
	t.Helper()

	sigs := make([][]uint32, len(blobs))

	for i, data := range blobs {
		sig, err := computer.ComputeSignature(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("signature %d: %v", i, err)
		}

		sigs[i] = sig
	}

	return sigs
}

func TestLSHBandingSelectorAssignsSimilarBlobs(t *testing.T) {
	computer := &GearCDCMinHashComputer{
		AvgChunkSize: 48,
		MinChunkSize: 16,
		MaxChunkSize: 256,
		K:            64,
	}

	// Blob 0: original
	original := make([]byte, 1024)
	for i := range original {
		original[i] = byte(i * 7)
	}

	// Blob 1: small edit of original (change 50 bytes)
	edited := make([]byte, len(original))
	copy(edited, original)
	for i := 500; i < 550; i++ {
		edited[i] = byte(i * 13)
	}

	// Blob 2: completely different
	unrelated := make([]byte, 1024)
	for i := range unrelated {
		unrelated[i] = byte(i*17 + 128)
	}

	blobData := [][]byte{original, edited, unrelated}
	sigs := makeTestSignatures(t, blobData, computer)

	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Size: 1024, Signature: sigs[0]},
			{Size: 1024, Signature: sigs[1]},
			{Size: 1024, Signature: sigs[2]},
		},
	}

	selector := &LSHBandingSelector{
		Bands:       16,
		RowsPerBand: 4,
		MinBlobSize: 100,
		MaxBlobSize: 10000,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	// Blob 0 and blob 1 should be paired (similar)
	// One should be assigned to the other
	assigned0, has0 := assignments.assignments[0]
	assigned1, has1 := assignments.assignments[1]

	paired := (has0 && assigned0 == 1) || (has1 && assigned1 == 0)
	if !paired {
		t.Error("expected blob 0 and 1 to be paired as similar")
	}
}

func TestLSHBandingSelectorSkipsNilSignatures(t *testing.T) {
	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Size: 1024, Signature: nil},
			{Size: 1024, Signature: nil},
		},
	}

	selector := &LSHBandingSelector{
		Bands:       16,
		RowsPerBand: 4,
		MinBlobSize: 100,
		MaxBlobSize: 10000,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	if len(assignments.assignments) != 0 {
		t.Errorf("expected no assignments for nil signatures, got %d",
			len(assignments.assignments))
	}
}

func TestLSHBandingSelectorRespectsMinBlobSize(t *testing.T) {
	sig := make([]uint32, 64)
	for i := range sig {
		sig[i] = uint32(i)
	}

	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Size: 10, Signature: sig},
			{Size: 10, Signature: sig},
		},
	}

	selector := &LSHBandingSelector{
		Bands:       16,
		RowsPerBand: 4,
		MinBlobSize: 100,
		MaxBlobSize: 10000,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	if len(assignments.assignments) != 0 {
		t.Errorf("expected no assignments for small blobs, got %d",
			len(assignments.assignments))
	}
}

func TestLSHBandingSelectorRespectsMaxBlobSize(t *testing.T) {
	sig := make([]uint32, 64)
	for i := range sig {
		sig[i] = uint32(i)
	}

	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Size: 50000, Signature: sig},
			{Size: 50000, Signature: sig},
		},
	}

	selector := &LSHBandingSelector{
		Bands:       16,
		RowsPerBand: 4,
		MinBlobSize: 100,
		MaxBlobSize: 10000,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	if len(assignments.assignments) != 0 {
		t.Errorf("expected no assignments for large blobs, got %d",
			len(assignments.assignments))
	}
}

func TestLSHBandingSelectorNoSelfAssignment(t *testing.T) {
	computer := &GearCDCMinHashComputer{
		AvgChunkSize: 48,
		MinChunkSize: 16,
		MaxChunkSize: 256,
		K:            64,
	}

	data := make([]byte, 512)
	for i := range data {
		data[i] = byte(i * 3)
	}

	// Two identical blobs
	sigs := makeTestSignatures(t, [][]byte{data, data}, computer)

	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Size: 512, Signature: sigs[0]},
			{Size: 512, Signature: sigs[1]},
		},
	}

	selector := &LSHBandingSelector{
		Bands:       16,
		RowsPerBand: 4,
		MinBlobSize: 100,
		MaxBlobSize: 10000,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	for blobIdx, baseIdx := range assignments.assignments {
		if blobIdx == baseIdx {
			t.Errorf("blob %d assigned to itself", blobIdx)
		}
	}
}

func TestLSHBandingSelectorSingleBlob(t *testing.T) {
	sig := make([]uint32, 64)
	blobs := &testBlobSet{
		blobs: []BlobMetadata{
			{Size: 1024, Signature: sig},
		},
	}

	selector := &LSHBandingSelector{
		Bands:       16,
		RowsPerBand: 4,
		MinBlobSize: 100,
		MaxBlobSize: 10000,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	if len(assignments.assignments) != 0 {
		t.Errorf("expected no assignments for single blob, got %d",
			len(assignments.assignments))
	}
}

func TestLSHBandingSelectorEmptyBlobSet(t *testing.T) {
	blobs := &testBlobSet{blobs: nil}

	selector := &LSHBandingSelector{
		Bands:       16,
		RowsPerBand: 4,
		MinBlobSize: 100,
		MaxBlobSize: 10000,
	}

	assignments := newTestAssignments()
	selector.SelectBases(blobs, assignments)

	if len(assignments.assignments) != 0 {
		t.Errorf("expected no assignments for empty set, got %d",
			len(assignments.assignments))
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd go && go test -v -tags test,debug -run TestLSHBanding ./src/echo/inventory_archive/`
Expected: compilation error — `LSHBandingSelector` undefined

**Step 3: Implement LSH banding selector**

Create `go/src/echo/inventory_archive/base_selector_lsh.go`:

```go
package inventory_archive

import (
	"hash/fnv"
	"math"
)

// LSHBandingSelector finds similar blobs via Locality-Sensitive Hashing
// over MinHash signatures stored in BlobMetadata.Signature. It divides
// each signature into Bands bands of RowsPerBand rows and hashes each
// band into a bucket. Blobs sharing any bucket are candidates. The best
// candidate (highest estimated Jaccard) becomes the delta base.
type LSHBandingSelector struct {
	Bands       int
	RowsPerBand int
	MinBlobSize uint64
	MaxBlobSize uint64
}

var _ BaseSelector = &LSHBandingSelector{}

func (s *LSHBandingSelector) SelectBases(
	blobs BlobSet,
	assignments DeltaAssignments,
) {
	n := blobs.Len()
	if n < 2 {
		return
	}

	expectedSigLen := s.Bands * s.RowsPerBand

	// Filter to eligible blobs (right size, has signature of correct length).
	type eligible struct {
		originalIndex int
		signature     []uint32
		size          uint64
	}

	var pool []eligible

	for i := range n {
		meta := blobs.At(i)

		if meta.Size < s.MinBlobSize || meta.Size > s.MaxBlobSize {
			continue
		}

		if len(meta.Signature) != expectedSigLen {
			continue
		}

		pool = append(pool, eligible{
			originalIndex: i,
			signature:     meta.Signature,
			size:          meta.Size,
		})
	}

	if len(pool) < 2 {
		return
	}

	// Build LSH band tables: band index -> band hash -> list of pool indices.
	type bucketKey struct {
		band int
		hash uint64
	}

	buckets := make(map[bucketKey][]int)

	for poolIdx, e := range pool {
		for b := range s.Bands {
			bandStart := b * s.RowsPerBand
			bandEnd := bandStart + s.RowsPerBand
			bh := hashBand(e.signature[bandStart:bandEnd])

			key := bucketKey{band: b, hash: bh}
			buckets[key] = append(buckets[key], poolIdx)
		}
	}

	// For each blob, find candidates and pick the best base.
	for poolIdx, e := range pool {
		candidates := make(map[int]bool)

		for b := range s.Bands {
			bandStart := b * s.RowsPerBand
			bandEnd := bandStart + s.RowsPerBand
			bh := hashBand(e.signature[bandStart:bandEnd])

			key := bucketKey{band: b, hash: bh}

			for _, otherIdx := range buckets[key] {
				if otherIdx != poolIdx {
					candidates[otherIdx] = true
				}
			}
		}

		if len(candidates) == 0 {
			continue
		}

		bestIdx := -1
		bestSim := -1.0

		for candIdx := range candidates {
			sim := MinHashJaccard(e.signature, pool[candIdx].signature)
			if sim > bestSim {
				bestSim = sim
				bestIdx = candIdx
			}
		}

		if bestIdx >= 0 && bestSim > 0 {
			// Assign smaller blob as delta against larger (or first against second
			// if equal). The base should ideally be the larger blob.
			basePoolIdx := bestIdx
			blobPoolIdx := poolIdx

			if pool[blobPoolIdx].size > pool[basePoolIdx].size {
				// Don't assign a larger blob as delta of a smaller one.
				// Swap: the current blob becomes the potential base,
				// but we only assign if the other hasn't been assigned yet.
				// Skip — the other blob will find us as a candidate and
				// make the assignment in the correct direction.
				continue
			}

			assignments.Assign(
				pool[blobPoolIdx].originalIndex,
				pool[basePoolIdx].originalIndex,
			)
		}
	}
}

func hashBand(rows []uint32) uint64 {
	h := fnv.New64a()

	for _, r := range rows {
		b := [4]byte{
			byte(r),
			byte(r >> 8),
			byte(r >> 16),
			byte(r >> 24),
		}
		h.Write(b[:])
	}

	return h.Sum64()
}
```

**Step 4: Run tests to verify they pass**

Run: `cd go && go test -v -tags test,debug -run TestLSHBanding ./src/echo/inventory_archive/`
Expected: all PASS

**Step 5: Run all inventory_archive tests**

Run: `cd go && go test -v -tags test,debug ./src/echo/inventory_archive/`
Expected: all PASS

**Step 6: Commit**

```
feat: add LSHBandingSelector for content-similarity base selection

Uses Locality-Sensitive Hashing over MinHash signatures to find similar
blob pairs in sublinear time. Divides signatures into bands, hashes
each band into buckets, and picks the best candidate by estimated
Jaccard similarity.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

---

### Task 5: Config Extensions

**Files:**
- Modify: `go/src/golf/blob_store_configs/main.go` (add new config interfaces)
- Modify: `go/src/golf/blob_store_configs/toml_inventory_archive_v1.go` (add signature + selector config)
- Modify: `go/src/golf/blob_store_configs/toml_inventory_archive_v2.go` (add signature + selector config)

**Step 1: Add config interfaces to `main.go`**

Add after the `DeltaConfigImmutable` interface block (after line 76):

```go
	SignatureConfigImmutable interface {
		GetSignatureType() string
		GetSignatureLen() int
		GetAvgChunkSize() int
		GetMinChunkSize() int
		GetMaxChunkSize() int
	}

	SelectorConfigImmutable interface {
		GetSelectorType() string
		GetSelectorBands() int
		GetSelectorRowsPerBand() int
		GetSelectorMinBlobSize() uint64
		GetSelectorMaxBlobSize() uint64
	}
```

**Step 2: Add config structs to `toml_inventory_archive_v1.go`**

Add `SignatureConfig` and `SelectorConfig` structs and embed them in `DeltaConfig`:

```go
type SignatureConfig struct {
	Type         string `toml:"type"`
	SignatureLen int    `toml:"signature-len"`
	AvgChunkSize int   `toml:"avg-chunk-size"`
	MinChunkSize int   `toml:"min-chunk-size"`
	MaxChunkSize int   `toml:"max-chunk-size"`
}

type SelectorConfig struct {
	Type        string `toml:"type"`
	Bands       int    `toml:"bands"`
	RowsPerBand int    `toml:"rows-per-band"`
	MinBlobSize uint64 `toml:"min-blob-size"`
	MaxBlobSize uint64 `toml:"max-blob-size"`
}
```

Add fields to `DeltaConfig`:

```go
type DeltaConfig struct {
	Enabled     bool            `toml:"enabled"`
	Algorithm   string          `toml:"algorithm"`
	MinBlobSize uint64          `toml:"min-blob-size"`
	MaxBlobSize uint64          `toml:"max-blob-size"`
	SizeRatio   float64         `toml:"size-ratio"`
	Signature   SignatureConfig `toml:"signature"`
	Selector    SelectorConfig  `toml:"selector"`
}
```

Add getter methods on `TomlInventoryArchiveV1` for both new interfaces. Do the
same for `TomlInventoryArchiveV2`. Add interface satisfaction assertions:

```go
var (
	_ SignatureConfigImmutable = TomlInventoryArchiveV1{}
	_ SelectorConfigImmutable  = TomlInventoryArchiveV1{}
)
```

**Step 3: Verify compilation**

Run: `cd go && go build ./src/golf/blob_store_configs/`
Expected: success

**Step 4: Verify all existing tests pass**

Run: `cd go && go test -v -tags test,debug ./src/echo/inventory_archive/ ./src/golf/blob_store_configs/`
Expected: all PASS

**Step 5: Commit**

```
feat: add signature and selector config for delta similarity

Extend DeltaConfig with nested SignatureConfig and SelectorConfig.
Add SignatureConfigImmutable and SelectorConfigImmutable interfaces.
Implement on both TomlInventoryArchiveV1 and V2.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

---

### Task 6: Signature + Selector Registries

**Files:**
- Create: `go/src/echo/inventory_archive/signature_registry.go`
- Create: `go/src/echo/inventory_archive/selector_registry.go`

**Step 1: Create signature registry**

Create `go/src/echo/inventory_archive/signature_registry.go`:

```go
package inventory_archive

import (
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/golf/blob_store_configs"
)

var signatureComputers = map[string]func(
	blob_store_configs.SignatureConfigImmutable,
) SignatureComputer{}

func RegisterSignatureComputer(
	name string,
	factory func(blob_store_configs.SignatureConfigImmutable) SignatureComputer,
) {
	signatureComputers[name] = factory
}

func SignatureComputerForConfig(
	config blob_store_configs.SignatureConfigImmutable,
) (SignatureComputer, error) {
	name := config.GetSignatureType()
	if name == "" {
		return nil, nil
	}

	factory, ok := signatureComputers[name]
	if !ok {
		return nil, errors.Errorf("unknown signature computer type: %q", name)
	}

	return factory(config), nil
}

func init() {
	RegisterSignatureComputer(
		"gear-cdc-minhash",
		func(config blob_store_configs.SignatureConfigImmutable) SignatureComputer {
			return &GearCDCMinHashComputer{
				AvgChunkSize: config.GetAvgChunkSize(),
				MinChunkSize: config.GetMinChunkSize(),
				MaxChunkSize: config.GetMaxChunkSize(),
				K:            config.GetSignatureLen(),
			}
		},
	)
}
```

**Step 2: Create selector registry**

Create `go/src/echo/inventory_archive/selector_registry.go`:

```go
package inventory_archive

import (
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/golf/blob_store_configs"
)

var baseSelectors = map[string]func(
	blob_store_configs.SelectorConfigImmutable,
) BaseSelector{}

func RegisterBaseSelector(
	name string,
	factory func(blob_store_configs.SelectorConfigImmutable) BaseSelector,
) {
	baseSelectors[name] = factory
}

func BaseSelectorForConfig(
	config blob_store_configs.SelectorConfigImmutable,
	deltaConfig blob_store_configs.DeltaConfigImmutable,
) (BaseSelector, error) {
	name := config.GetSelectorType()

	if name == "" || name == "size-based" {
		return &SizeBasedSelector{
			MinBlobSize: deltaConfig.GetDeltaMinBlobSize(),
			MaxBlobSize: deltaConfig.GetDeltaMaxBlobSize(),
			SizeRatio:   deltaConfig.GetDeltaSizeRatio(),
		}, nil
	}

	factory, ok := baseSelectors[name]
	if !ok {
		return nil, errors.Errorf("unknown base selector type: %q", name)
	}

	return factory(config), nil
}

func init() {
	RegisterBaseSelector(
		"lsh-banding",
		func(config blob_store_configs.SelectorConfigImmutable) BaseSelector {
			return &LSHBandingSelector{
				Bands:       config.GetSelectorBands(),
				RowsPerBand: config.GetSelectorRowsPerBand(),
				MinBlobSize: config.GetSelectorMinBlobSize(),
				MaxBlobSize: config.GetSelectorMaxBlobSize(),
			}
		},
	)
}
```

**Step 3: Verify compilation**

Run: `cd go && go build ./src/echo/inventory_archive/`
Expected: success

**Step 4: Commit**

```
feat: add signature computer and base selector registries

Map type strings to factory functions, following the same pattern as
DeltaAlgorithm. Register gear-cdc-minhash and lsh-banding as built-in
implementations. Default selector falls back to size-based.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

---

### Task 7: Wire Packer to Use Config-Driven Selector

**Files:**
- Modify: `go/src/india/blob_stores/pack_v1.go:257-312` (signature phase + config-driven selector)

**Step 1: Update `packChunkArchiveV1` to use registries**

In `pack_v1.go`, replace the hardcoded `SizeBasedSelector` construction
(lines 289-311) with config-driven selector and optional signature computation.

The existing code at lines 289-311:

```go
		blobSet := &sliceBlobSet{...}
		for i, blob := range blobs { ... }

		selector := &inventory_archive.SizeBasedSelector{
			MinBlobSize: store.config.GetDeltaMinBlobSize(),
			MaxBlobSize: store.config.GetDeltaMaxBlobSize(),
			SizeRatio:   store.config.GetDeltaSizeRatio(),
		}

		da := &mapDeltaAssignments{assignments: assignments}
		selector.SelectBases(blobSet, da)
```

Replace with:

```go
		// Resolve selector from config.
		deltaConfig, ok := store.config.(blob_store_configs.ConfigInventoryArchiveDelta)
		if !ok {
			err = errors.Errorf("config does not implement ConfigInventoryArchiveDelta")
			return dataPath, 0, 0, err
		}

		sigConfig, hasSigConfig := store.config.(blob_store_configs.SignatureConfigImmutable)
		selConfig, hasSelConfig := store.config.(blob_store_configs.SelectorConfigImmutable)

		var selector inventory_archive.BaseSelector

		if hasSelConfig {
			var selErr error
			selector, selErr = inventory_archive.BaseSelectorForConfig(selConfig, deltaConfig)
			if selErr != nil {
				err = errors.Wrap(selErr)
				return dataPath, 0, 0, err
			}
		} else {
			selector = &inventory_archive.SizeBasedSelector{
				MinBlobSize: deltaConfig.GetDeltaMinBlobSize(),
				MaxBlobSize: deltaConfig.GetDeltaMaxBlobSize(),
				SizeRatio:   deltaConfig.GetDeltaSizeRatio(),
			}
		}

		// Build BlobSet with optional signatures.
		blobSet := &sliceBlobSet{
			blobs: make([]inventory_archive.BlobMetadata, len(blobs)),
		}

		for i, blob := range blobs {
			marklId, repool := store.defaultHash.GetBlobIdForHexString(
				hex.EncodeToString(blob.digest),
			)
			blobSet.blobs[i] = inventory_archive.BlobMetadata{
				Id:   marklId,
				Size: uint64(len(blob.data)),
			}
			repool()
		}

		// Compute signatures if configured.
		if hasSigConfig {
			sigComputer, sigErr := inventory_archive.SignatureComputerForConfig(sigConfig)
			if sigErr != nil {
				err = errors.Wrap(sigErr)
				return dataPath, 0, 0, err
			}

			if sigComputer != nil {
				for i, blob := range blobs {
					sig, compErr := sigComputer.ComputeSignature(
						bytes.NewReader(blob.data),
					)
					if compErr != nil {
						err = errors.Wrapf(compErr, "computing signature for blob %d", i)
						return dataPath, 0, 0, err
					}

					blobSet.blobs[i].Signature = sig
				}
			}
		}

		da := &mapDeltaAssignments{assignments: assignments}
		selector.SelectBases(blobSet, da)
```

Note: `store.config` already implements `ConfigInventoryArchiveDelta` (asserted
in the existing code path). After Task 5, it also implements
`SignatureConfigImmutable` and `SelectorConfigImmutable`.

**Step 2: Verify compilation**

Run: `cd go && go build ./src/india/blob_stores/`
Expected: success

**Step 3: Run full test suite**

Run: `cd go && just test-go`
Expected: all PASS

**Step 4: Commit**

```
feat: wire packer to use config-driven signature computer and selector

Replace hardcoded SizeBasedSelector in packChunkArchiveV1 with
registry-based lookup. Compute signatures when configured before
calling selector.

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

---

### Task 8: Full Build and Integration Test

**Step 1: Build**

Run: `just build`
Expected: success

**Step 2: Run unit tests**

Run: `just test-go`
Expected: all PASS

**Step 3: Run integration tests**

Run: `just test-bats`
Expected: all PASS (existing delta tests use size-based selector by default)

**Step 4: Final commit if any fixups needed**

---

## Deferred Items (not in this plan)

- Packer parallelization of signature computation (worker pool)
- BATS integration tests specifically exercising LSH selector config
- Benchmarks comparing size-based vs LSH selector
- Pre-computed signatures at blob ingest time
  (see `docs/plans/2026-02-23-dynamic-type-registries.md`)
- Dynamic loading of signature computers and selectors via plugins
  (see `docs/plans/2026-02-23-dynamic-type-registries.md`)
