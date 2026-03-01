package stream_index_fixed

import (
	"bytes"
	"os"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/echo/genesis_configs"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/object_finalizer"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

func TestFixedBinaryRoundTrip(t1 *testing.T) {
	t := ui.T{T: t1}

	encoder := binaryEncoder{Sigil: ids.SigilLatest}
	expected, _ := sku.GetTransactedPool().GetWithRepool()

	{
		t.AssertNoError(
			expected.GetObjectIdMutable().SetWithId(ids.MustZettelId("one/uno")),
		)
		expected.SetTai(ids.NowTai())
		t.AssertNoError(markl.SetHexBytes(
			markl.FormatIdHashSha256,
			expected.GetMetadataMutable().GetBlobDigestMutable(),
			[]byte(
				"ed500e315f33358824203cee073893311e0a80d77989dc55c5d86247d95b2403",
			),
		))

		metadata := expected.GetMetadataMutable()
		t.AssertNoError(metadata.GetTypeMutable().SetType("!da-typ"))

		{
			typeSig := metadata.GetTypeLockMutable()
			t.AssertNoError(typeSig.GetValueMutable().GeneratePrivateKey(
				nil,
				markl.FormatIdNonceSec,
				"",
			))
		}

		t.AssertNoError(metadata.GetDescriptionMutable().Set("the bez"))
		t.AssertNoError(expected.AddTag(ids.MustTag("tag")))

		{
			config := genesis_configs.Default().Blob
			finalizer := object_finalizer.Make()

			t.AssertNoError(config.GetPrivateKeyMutable().GeneratePrivateKey(
				nil,
				markl.FormatIdEd25519Sec,
				markl.PurposeRepoPrivateKeyV1,
			))
			t.AssertNoError(finalizer.FinalizeAndSignOverwrite(expected, config))
		}
	}

	// Create a temp file for overflow (signed objects usually exceed 255 bytes).
	overflowTempFile, err := os.CreateTemp("", "test-overflow-*")
	if err != nil {
		t.Fatalf("failed to create overflow temp file: %s", err)
	}

	defer os.Remove(overflowTempFile.Name())
	defer overflowTempFile.Close()

	overflowW := makeOverflowWriterForTempFile(overflowTempFile)

	// Encode to fixed-length entry.
	var entryBuf bytes.Buffer

	if err := encoder.writeFixedEntry(
		&entryBuf,
		objectWithSigil{Transacted: expected, Sigil: ids.SigilLatest},
		overflowW,
	); err != nil {
		t.Fatalf("encode failed: %s", err)
	}

	if entryBuf.Len() != EntryWidth {
		t.Fatalf("expected %d bytes, got %d", EntryWidth, entryBuf.Len())
	}

	entryBytes := entryBuf.Bytes()

	// Verify sigil at byte offset 3 (raw byte value, not character encoding).
	decodedSigil := ids.Sigil(entryBytes[3])
	if !decodedSigil.Contains(ids.SigilLatest) {
		t.Fatalf("expected SigilLatest at byte 3, got %x", entryBytes[3])
	}

	hasOverflow := entryBytes[EntryWidth-1]&0x01 != 0
	t.Logf("entry has overflow: %v", hasOverflow)

	// Decode the entry.
	decoder := makeBinaryDecoder(ids.SigilLatest)
	actual, _ := sku.GetTransactedPool().GetWithRepool()

	entryReader := bytes.NewReader(entryBytes)
	overflowRA := &overflowReaderAt{readerAt: overflowTempFile}

	if err := decoder.readFixedEntry(
		entryReader,
		overflowRA,
		0,
		actual,
	); err != nil {
		t.Fatalf("decode failed: %s", err)
	}

	if !sku.TransactedEqualer.Equals(expected, actual) {
		t.Errorf(
			"round-trip mismatch:\nexpected: %q\nactual:   %q",
			sku.String(expected),
			sku.String(actual),
		)
	}

	// Verify in-place sigil update.
	{
		entryData := make([]byte, EntryWidth)
		copy(entryData, entryBytes)

		writerAt := makeBytesWriterAt(entryData)

		var updateSigil ids.Sigil
		updateSigil.Add(ids.SigilHistory)
		updateSigil.Add(ids.SigilLatest)

		if err := encoder.updateSigil(writerAt, updateSigil, 0); err != nil {
			t.Fatalf("updateSigil failed: %s", err)
		}

		resultSigil := ids.Sigil(entryData[3])

		if !resultSigil.Contains(ids.SigilLatest) {
			t.Fatal("expected SigilLatest after update")
		}
	}
}

func TestFixedBinaryOverflow(t1 *testing.T) {
	t := ui.T{T: t1}

	encoder := binaryEncoder{Sigil: ids.SigilHistory}
	object, _ := sku.GetTransactedPool().GetWithRepool()

	t.AssertNoError(
		object.GetObjectIdMutable().SetWithId(ids.MustZettelId("one/uno")),
	)
	object.SetTai(ids.NowTai())
	t.AssertNoError(markl.SetHexBytes(
		markl.FormatIdHashSha256,
		object.GetMetadataMutable().GetBlobDigestMutable(),
		[]byte(
			"ed500e315f33358824203cee073893311e0a80d77989dc55c5d86247d95b2403",
		),
	))

	metadata := object.GetMetadataMutable()
	t.AssertNoError(metadata.GetTypeMutable().SetType("!da-typ"))

	{
		typeSig := metadata.GetTypeLockMutable()
		t.AssertNoError(typeSig.GetValueMutable().GeneratePrivateKey(
			nil,
			markl.FormatIdNonceSec,
			"",
		))
	}

	t.AssertNoError(metadata.GetDescriptionMutable().Set("the bez"))
	t.AssertNoError(object.AddTag(ids.MustTag("tag")))

	{
		config := genesis_configs.Default().Blob
		finalizer := object_finalizer.Make()

		t.AssertNoError(config.GetPrivateKeyMutable().GeneratePrivateKey(
			nil,
			markl.FormatIdEd25519Sec,
			markl.PurposeRepoPrivateKeyV1,
		))
		t.AssertNoError(finalizer.FinalizeAndSignOverwrite(object, config))
	}

	// Add many tags to push into overflow.
	for i := range 20 {
		tagName := ids.MustTag(
			t1.Name() + "-" + string(rune('a'+i)) + "-long-tag-name-to-fill-space",
		)
		t.AssertNoError(object.AddTag(tagName))
	}

	// Check if the field data exceeds InlineCapacity by doing a dry-run encode.
	encoder.fieldBuffer.Reset()
	for _, key := range fixedFieldOrder {
		encoder.field.Reset()
		encoder.field.Key = key
		encoder.writeFieldKey(objectWithSigil{Transacted: object})
	}

	fieldLen := encoder.fieldBuffer.Len()
	t.Logf("field data length: %d (InlineCapacity: %d)", fieldLen, InlineCapacity)

	if fieldLen <= InlineCapacity {
		t1.Skip("test object did not exceed InlineCapacity - adjust tags")
	}

	t.Logf(
		"overflow triggered: %d bytes field data > %d InlineCapacity",
		fieldLen,
		InlineCapacity,
	)
}

func TestFixedBinaryZeroPadding(t1 *testing.T) {
	t := ui.T{T: t1}

	encoder := binaryEncoder{Sigil: ids.SigilHistory}
	object, _ := sku.GetTransactedPool().GetWithRepool()

	// Minimal object — should have significant zero padding.
	t.AssertNoError(
		object.GetObjectIdMutable().SetWithId(ids.MustZettelId("a/b")),
	)
	object.SetTai(ids.NowTai())

	var entryBuf bytes.Buffer

	if err := encoder.writeFixedEntry(
		&entryBuf,
		objectWithSigil{Transacted: object, Sigil: ids.SigilHistory},
		nil,
	); err != nil {
		t.Fatalf("encode failed: %s", err)
	}

	if entryBuf.Len() != EntryWidth {
		t.Fatalf("expected %d bytes, got %d", EntryWidth, entryBuf.Len())
	}

	entryBytes := entryBuf.Bytes()

	// Last byte should have overflow bit clear.
	if entryBytes[EntryWidth-1]&0x01 != 0 {
		t.Fatal("expected no overflow for minimal object")
	}

	// There should be zero padding between end of fields and last byte.
	zeroCount := 0
	for i := len(entryBytes) - 2; i >= 0; i-- {
		if entryBytes[i] == 0 {
			zeroCount++
		} else {
			break
		}
	}

	t.Logf("zero-padded bytes: %d", zeroCount)

	if zeroCount == 0 {
		t.Fatal("expected some zero padding for minimal object")
	}
}

// bytesWriterAt implements io.WriterAt over a byte slice.
type bytesWriterAt struct {
	data []byte
}

func makeBytesWriterAt(data []byte) *bytesWriterAt {
	return &bytesWriterAt{data: data}
}

func (w *bytesWriterAt) WriteAt(p []byte, off int64) (n int, err error) {
	copy(w.data[off:], p)
	return len(p), nil
}
