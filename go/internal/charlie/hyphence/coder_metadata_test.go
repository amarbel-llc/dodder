package hyphence

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
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

// https://github.com/amarbel-llc/dodder/issues/41
// Blob body immediately after closing --- (no blank line) should fail.

func TestReaderMissingNewlineBetweenBoundaryAndBlobShouldFail(t *testing.T) {
	body := "---\n! toml-type-v1\n---\nfile-extension = 'png'\n"

	var metadata, blob bytes.Buffer

	reader := Reader{
		Metadata: &metadata,
		Blob:     &blob,
	}

	_, err := reader.ReadFrom(strings.NewReader(body))
	if err == nil && blob.Len() == 0 {
		t.Fatal("expected error when blob body immediately follows closing --- without blank line separator, but blob was silently dropped")
	} else if err == nil {
		t.Fatalf("expected error but blob was parsed as: %q", blob.String())
	}
}

func TestDecoderMissingNewlineBetweenBoundaryAndBlobShouldFail(t *testing.T) {
	// No blank line between closing --- and blob body
	body := "---\n! toml-type-v1\n---\nfile-extension = 'png'\n"

	var blobCapture bytes.Buffer

	decoder := Decoder[*TypedBlob[struct{}]]{
		Metadata:      TypedMetadataCoder[struct{}]{},
		Blob:          testBlobDecoder{},
		BlobTeeWriter: &blobCapture,
	}

	typedBlob := &TypedBlob[struct{}]{}
	reader := bufio.NewReader(strings.NewReader(body))

	_, err := decoder.DecodeFrom(typedBlob, reader)
	if err == nil {
		t.Fatal("expected error when blob body immediately follows closing --- without blank line separator")
	}
}

func TestReaderAllowMissingSeparatorForwardsBlobContent(t *testing.T) {
	body := "---\n! toml-type-v1\n---\nfile-extension = 'png'\n"

	var metadata, blob bytes.Buffer

	reader := Reader{
		Metadata:              &metadata,
		Blob:                  &blob,
		AllowMissingSeparator: true,
	}

	if _, err := reader.ReadFrom(strings.NewReader(body)); err != nil {
		t.Fatal(err)
	}

	if blob.String() != "file-extension = 'png'\n" {
		t.Fatalf("expected blob content forwarded, got %q", blob.String())
	}
}

func TestDecoderAllowMissingSeparatorForwardsBlobContent(t *testing.T) {
	body := "---\n! toml-type-v1\n---\nfile-extension = 'png'\n"

	var blobCapture bytes.Buffer

	decoder := Decoder[*TypedBlob[struct{}]]{
		Metadata:              TypedMetadataCoder[struct{}]{},
		Blob:                  testBlobDecoder{},
		BlobTeeWriter:         &blobCapture,
		AllowMissingSeparator: true,
	}

	typedBlob := &TypedBlob[struct{}]{}
	reader := bufio.NewReader(strings.NewReader(body))

	if _, err := decoder.DecodeFrom(typedBlob, reader); err != nil {
		t.Fatal(err)
	}

	if blobCapture.String() != "file-extension = 'png'\n" {
		t.Fatalf("expected blob content forwarded, got %q", blobCapture.String())
	}
}

func TestWriterOutputParsesWithStrictReader(t *testing.T) {
	var metadataContent bytes.Buffer
	metadataContent.WriteString("! toml-type-v1\n")

	var blobContent bytes.Buffer
	blobContent.WriteString("file-extension = 'png'\n")

	writer := Writer{
		Metadata: &metadataContent,
		Blob:     &blobContent,
	}

	var output bytes.Buffer
	if _, err := writer.WriteTo(&output); err != nil {
		t.Fatal(err)
	}

	t.Logf("Writer output: %q", output.String())

	// Now parse with strict Reader
	var parsedMetadata, parsedBlob bytes.Buffer
	reader := Reader{
		Metadata: &parsedMetadata,
		Blob:     &parsedBlob,
	}

	if _, err := reader.ReadFrom(strings.NewReader(output.String())); err != nil {
		t.Fatalf("strict Reader failed on Writer output: %v", err)
	}

	if parsedBlob.String() != "file-extension = 'png'\n" {
		t.Fatalf("blob mismatch: got %q", parsedBlob.String())
	}
}

type testBlobEncoder struct{}

func (testBlobEncoder) EncodeTo(
	_ *TypedBlob[struct{}],
	writer *bufio.Writer,
) (n int64, err error) {
	return 0, nil
}

func TestEncoderOutputParsesWithStrictDecoder(t *testing.T) {
	original := &TypedBlob[struct{}]{}
	if err := original.Type.Set("toml-type-v1"); err != nil {
		t.Fatal(err)
	}

	// Encode
	var buf bytes.Buffer
	encoder := Encoder[*TypedBlob[struct{}]]{
		Metadata: TypedMetadataCoder[struct{}]{},
		Blob:     testBlobEncoder{},
	}

	if _, err := encoder.EncodeTo(original, &buf); err != nil {
		t.Fatal(err)
	}

	t.Logf("Encoder output: %q", buf.String())

	// Decode with strict Decoder
	decoded := &TypedBlob[struct{}]{}
	decoder := Decoder[*TypedBlob[struct{}]]{
		Metadata: TypedMetadataCoder[struct{}]{},
		Blob:     testBlobDecoder{},
	}

	reader := bufio.NewReader(strings.NewReader(buf.String()))
	if _, err := decoder.DecodeFrom(decoded, reader); err != nil {
		t.Fatalf("strict Decoder failed on Encoder output: %v", err)
	}

	if decoded.Type.String() != original.Type.String() {
		t.Fatalf("type mismatch: got %q, want %q", decoded.Type, original.Type)
	}
}

// Verify that the TypedMetadataCoder implements the expected interface.
var _ interfaces.CoderBufferedReadWriter[*TypedBlob[struct{}]] = TypedMetadataCoder[struct{}]{}
