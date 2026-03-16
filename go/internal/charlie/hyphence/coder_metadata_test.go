package hyphence

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

func TestTypedMetadataCoderRoundtripWithBlobDigest(t *testing.T) {
	var blobDigest markl.Id
	if err := blobDigest.Set(
		"blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0",
	); err != nil {
		t.Fatal(err)
	}

	original := &TypedBlob[struct{}]{
		BlobDigest: blobDigest,
	}
	if err := original.Type.Set("inventory_list-v2"); err != nil {
		t.Fatal(err)
	}

	// Encode
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	coder := TypedMetadataCoder[struct{}]{}

	if _, err := coder.EncodeTo(original, writer); err != nil {
		t.Fatal(err)
	}
	if err := writer.Flush(); err != nil {
		t.Fatal(err)
	}

	encoded := buf.String()
	if !strings.Contains(encoded, "! inventory_list-v2") {
		t.Fatalf("expected type line in encoded output: %q", encoded)
	}
	if !strings.Contains(encoded, "@ blake2b256-") {
		t.Fatalf("expected blob digest line in encoded output: %q", encoded)
	}

	// Decode
	decoded := &TypedBlob[struct{}]{}
	reader := bufio.NewReader(strings.NewReader(encoded))

	if _, err := coder.DecodeFrom(decoded, reader); err != nil {
		t.Fatal(err)
	}

	if decoded.Type.String() != original.Type.String() {
		t.Fatalf("type mismatch: got %q, want %q", decoded.Type, original.Type)
	}

	if decoded.BlobDigest.String() != original.BlobDigest.String() {
		t.Fatalf(
			"blob digest mismatch: got %q, want %q",
			decoded.BlobDigest.String(),
			original.BlobDigest.String(),
		)
	}
}

func TestTypedMetadataCoderOmitsNullBlobDigest(t *testing.T) {
	original := &TypedBlob[struct{}]{}
	if err := original.Type.Set("inventory_list-v2"); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	coder := TypedMetadataCoder[struct{}]{}

	if _, err := coder.EncodeTo(original, writer); err != nil {
		t.Fatal(err)
	}
	if err := writer.Flush(); err != nil {
		t.Fatal(err)
	}

	encoded := buf.String()
	if strings.Contains(encoded, "@") {
		t.Fatalf("null blob digest should not be encoded: %q", encoded)
	}
}

type testBlobDecoder struct{}

func (testBlobDecoder) DecodeFrom(
	_ *TypedBlob[struct{}],
	reader *bufio.Reader,
) (n int64, err error) {
	data, err := reader.ReadString(0)
	if err != nil {
		n += int64(len(data))
	}

	return n, nil
}

func TestDecoderBlobTeeWriterCapturesBlobContent(t *testing.T) {
	body := "---\n! inventory_list-v2\n@ blake2b256-c5xgv9eyuv6g49mcwqks24gd3dh39w8220l0kl60qxt60rnt60lsc8fqv0\n---\n\nhello blob content"

	var blobCapture bytes.Buffer

	decoder := Decoder[*TypedBlob[struct{}]]{
		Metadata:      TypedMetadataCoder[struct{}]{},
		Blob:          testBlobDecoder{},
		BlobTeeWriter: &blobCapture,
	}

	typedBlob := &TypedBlob[struct{}]{}
	reader := bufio.NewReader(strings.NewReader(body))

	if _, err := decoder.DecodeFrom(typedBlob, reader); err != nil {
		t.Fatal(err)
	}

	if typedBlob.Type.String() != "!inventory_list-v2" {
		t.Fatalf("type not decoded: got %q", typedBlob.Type.String())
	}

	if typedBlob.BlobDigest.IsNull() {
		t.Fatal("blob digest was not decoded from metadata")
	}

	captured := blobCapture.String()
	if captured != "hello blob content" {
		t.Fatalf(
			"BlobTeeWriter should capture only blob content, got %q",
			captured,
		)
	}
}

// Verify that the TypedMetadataCoder implements the expected interface.
var _ interfaces.CoderBufferedReadWriter[*TypedBlob[struct{}]] = TypedMetadataCoder[struct{}]{}
