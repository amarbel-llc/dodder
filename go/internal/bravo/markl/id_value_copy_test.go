package markl

import (
	"io"
	"testing"

	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

func TestIdValueCopyPreservesData(t1 *testing.T) {
	ui.RunTestContext(t1, testIdValueCopyPreservesData)
}

func testIdValueCopyPreservesData(t *ui.TestContext) {
	// Create a real hash-based markl.Id (simulating typeObject.GetBlobDigest())
	formatHash := FormatHashBlake2b256
	hash, hashRepool := formatHash.Get()
	defer hashRepool()

	if _, err := io.WriteString(hash, "test blob content"); err != nil {
		t.AssertNoError(err)
	}

	sourceId, sourceRepool := hash.GetMarklId()
	defer sourceRepool()

	t.Logf("source: string=%s isEmpty=%t isNull=%t bytes=%x",
		sourceId.String(), sourceId.IsEmpty(), sourceId.IsNull(), sourceId.GetBytes())

	// Step 1: ResetWithMarklId into a fresh Id (simulates field_reader.go line 142-143)
	var copied Id
	copied.ResetWithMarklId(sourceId)

	t.Logf("after ResetWithMarklId: string=%s isEmpty=%t format=%v bytes=%x",
		copied.String(), copied.IsEmpty(), copied.GetMarklFormat(), copied.GetBytes())

	t.AssertFalse(copied.IsEmpty(), "copied should not be empty after ResetWithMarklId")

	// Step 2: Embed in a struct and copy by value (simulates Field{TypeBlobDigest: copied})
	type FieldLike struct {
		Key            string
		Value          string
		TypeBlobDigest Id
	}

	field := FieldLike{
		Key:            "status",
		Value:          "todo",
		TypeBlobDigest: copied,
	}

	t.Logf("after struct embed: isEmpty=%t format=%v bytes=%x",
		field.TypeBlobDigest.IsEmpty(), field.TypeBlobDigest.GetMarklFormat(), field.TypeBlobDigest.GetBytes())

	t.AssertFalse(field.TypeBlobDigest.IsEmpty(), "should not be empty after struct embed")

	// Step 3: Append to a slice (simulates collections_slice.Append)
	slice := make([]FieldLike, 0, 4)
	slice = append(slice, field)

	t.Logf("after slice append: isEmpty=%t format=%v bytes=%x",
		slice[0].TypeBlobDigest.IsEmpty(), slice[0].TypeBlobDigest.GetMarklFormat(), slice[0].TypeBlobDigest.GetBytes())

	t.AssertFalse(slice[0].TypeBlobDigest.IsEmpty(), "should not be empty after slice append")

	// Step 4: Iterate with range (simulates for field := range ...GetFields())
	for _, f := range slice {
		t.Logf("after range iterate: isEmpty=%t format=%v bytes=%x",
			f.TypeBlobDigest.IsEmpty(), f.TypeBlobDigest.GetMarklFormat(), f.TypeBlobDigest.GetBytes())

		t.AssertFalse(f.TypeBlobDigest.IsEmpty(), "should not be empty after range iteration")
	}
}

func TestIdResetWithMarklIdFromPooledHash(t1 *testing.T) {
	ui.RunTestContext(t1, testIdResetWithMarklIdFromPooledHash)
}

func testIdResetWithMarklIdFromPooledHash(t *ui.TestContext) {
	// Simulate the exact sequence in field_reader.go:
	// 1. typeObject.GetBlobDigest() returns a pooled markl.Id
	// 2. var typeBlobDigest markl.Id
	// 3. typeBlobDigest.ResetWithMarklId(typeObject.GetBlobDigest())
	// 4. The pooled source is repooled (format pointer may be invalidated)
	// 5. typeBlobDigest is used in a Field struct

	formatHash := FormatHashBlake2b256
	hash, hashRepool := formatHash.Get()

	if _, err := io.WriteString(hash, "pooled test"); err != nil {
		t.AssertNoError(err)
	}

	sourceId, sourceRepool := hash.GetMarklId()

	// Copy while source is still valid
	var typeBlobDigest Id
	typeBlobDigest.ResetWithMarklId(sourceId)

	t.Logf("before repool: string=%s isEmpty=%t bytes=%x",
		typeBlobDigest.String(), typeBlobDigest.IsEmpty(), typeBlobDigest.GetBytes())

	// Now repool the source (simulating what happens after ParseTypedBlob's defer repool())
	sourceRepool()
	hashRepool()

	// Check if our copy survived the repool
	t.Logf("after repool: isEmpty=%t format=%v bytes=%x",
		typeBlobDigest.IsEmpty(), typeBlobDigest.GetMarklFormat(), typeBlobDigest.GetBytes())

	t.AssertFalse(typeBlobDigest.IsEmpty(), "should not be empty after source repool")

	// Verify it's usable in a field
	type FieldLike struct {
		TypeBlobDigest Id
	}

	field := FieldLike{TypeBlobDigest: typeBlobDigest}
	t.AssertFalse(field.TypeBlobDigest.IsEmpty(), "should not be empty in field struct after repool")
}
