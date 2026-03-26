package repo_blobs

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

func TestCoderTommyLocalOverridePathV0_RoundTrip(t *testing.T) {
	coder := Coder.Blob[ids.TypeTomlRepoLocalOverridePath]
	if coder == nil {
		t.Fatal("no coder registered for TypeTomlRepoLocalOverridePath")
	}

	original := &TomlLocalOverridePathV0{
		OverridePath: "/test/path",
	}

	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	var blob Blob = original
	if _, err := coder.EncodeTo(&blob, writer); err != nil {
		t.Fatalf("EncodeTo failed: %v", err)
	}
	writer.Flush()
	encoded := buf.Bytes()

	if strings.Contains(string(encoded), "---") {
		t.Error("blob body contains hyphence header --- (should be added by higher layer)")
	}

	var decoded Blob
	reader := bufio.NewReader(bytes.NewReader(encoded))
	if _, err := coder.DecodeFrom(&decoded, reader); err != nil {
		t.Fatalf("DecodeFrom failed: %v", err)
	}

	got, ok := decoded.(*TomlLocalOverridePathV0)
	if !ok {
		t.Fatalf("decoded type = %T, want *TomlLocalOverridePathV0", decoded)
	}

	if got.OverridePath != original.OverridePath {
		t.Errorf("OverridePath = %q, want %q", got.OverridePath, original.OverridePath)
	}
}

func TestCoderTommyLocalOverridePathV0_EncodeDecodeEncode(t *testing.T) {
	coder := Coder.Blob[ids.TypeTomlRepoLocalOverridePath]
	if coder == nil {
		t.Fatal("no coder registered")
	}

	original := &TomlLocalOverridePathV0{
		OverridePath: "/test/path",
	}

	var buf1 bytes.Buffer
	writer1 := bufio.NewWriter(&buf1)
	var blob1 Blob = original
	if _, err := coder.EncodeTo(&blob1, writer1); err != nil {
		t.Fatalf("first EncodeTo failed: %v", err)
	}
	writer1.Flush()
	firstEncode := buf1.Bytes()

	var decoded Blob
	reader := bufio.NewReader(bytes.NewReader(firstEncode))
	if _, err := coder.DecodeFrom(&decoded, reader); err != nil {
		t.Fatalf("DecodeFrom failed: %v", err)
	}

	var buf2 bytes.Buffer
	writer2 := bufio.NewWriter(&buf2)
	if _, err := coder.EncodeTo(&decoded, writer2); err != nil {
		t.Fatalf("second EncodeTo failed: %v", err)
	}
	writer2.Flush()
	secondEncode := buf2.Bytes()

	if !bytes.Equal(firstEncode, secondEncode) {
		t.Errorf("encode-decode-encode not stable:\nfirst:  %q\nsecond: %q", string(firstEncode), string(secondEncode))
	}
}

func TestCoderTommyLocalOverridePathV0_IsCoderTommy(t *testing.T) {
	coder, ok := Coder.Blob[ids.TypeTomlRepoLocalOverridePath]
	if !ok {
		t.Fatal("TypeTomlRepoLocalOverridePath not in coder map")
	}

	if _, isTommy := coder.(hyphence.CoderTommy[Blob, *Blob]); !isTommy {
		t.Errorf("coder is %T, want CoderTommy", coder)
	}
}

var _ interfaces.CoderBufferedReadWriter[*Blob] = Coder.Blob[ids.TypeTomlRepoLocalOverridePath]
