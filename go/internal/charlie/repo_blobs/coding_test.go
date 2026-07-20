package repo_blobs

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/0/hyphence"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

func TestCoderTommyLocalOverridePathV0_RoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)
	coder := Coder.Blob[ids.TypeTomlRepoLocalOverridePath]
	t.AssertNotNil(coder, "no coder registered for TypeTomlRepoLocalOverridePath")

	original := &TomlLocalOverridePathV0{
		OverridePath: "/test/path",
	}

	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)
	var blob Blob = original
	_, err := coder.EncodeTo(&blob, writer)
	t.AssertNoError(err)
	writer.Flush()
	encoded := buf.Bytes()

	if strings.Contains(string(encoded), "---") {
		t.Error("blob body contains hyphence header --- (should be added by higher layer)")
	}

	var decoded Blob
	reader := bufio.NewReader(bytes.NewReader(encoded))
	_, err = coder.DecodeFrom(&decoded, reader)
	t.AssertNoError(err)

	got, ok := decoded.(*TomlLocalOverridePathV0)
	if !ok {
		t.Fatalf("decoded type = %T, want *TomlLocalOverridePathV0", decoded)
	}

	t.AssertEqualStrings(original.OverridePath, got.OverridePath)
}

func TestCoderTommyLocalOverridePathV0_EncodeDecodeEncode(t1 *testing.T) {
	t := ui.MakeT(t1)
	coder := Coder.Blob[ids.TypeTomlRepoLocalOverridePath]
	t.AssertNotNil(coder, "no coder registered")

	original := &TomlLocalOverridePathV0{
		OverridePath: "/test/path",
	}

	var buf1 bytes.Buffer
	writer1 := bufio.NewWriter(&buf1)
	var blob1 Blob = original
	_, err := coder.EncodeTo(&blob1, writer1)
	t.AssertNoError(err)
	writer1.Flush()
	firstEncode := buf1.Bytes()

	var decoded Blob
	reader := bufio.NewReader(bytes.NewReader(firstEncode))
	_, err = coder.DecodeFrom(&decoded, reader)
	t.AssertNoError(err)

	var buf2 bytes.Buffer
	writer2 := bufio.NewWriter(&buf2)
	_, err = coder.EncodeTo(&decoded, writer2)
	t.AssertNoError(err)
	writer2.Flush()
	secondEncode := buf2.Bytes()

	t.AssertEqual(firstEncode, secondEncode)
}

func TestCoderTommyLocalOverridePathV0_IsCoderTommy(t1 *testing.T) {
	t := ui.MakeT(t1)
	coder, ok := Coder.Blob[ids.TypeTomlRepoLocalOverridePath]
	if !ok {
		t.Fatal("TypeTomlRepoLocalOverridePath not in coder map")
	}

	if _, isTommy := coder.(hyphence.CoderTommy[Blob, *Blob]); !isTommy {
		t.Errorf("coder is %T, want CoderTommy", coder)
	}
}

var _ interfaces.CoderBufferedReadWriter[*Blob] = Coder.Blob[ids.TypeTomlRepoLocalOverridePath]
