package ids

import (
	"bytes"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

func idWriteToReadFromData() []string {
	return []string{
		"one/uno",
		"config",
		"!md",
		"-tag",
		"/repo",
		"tag",
	}
}

func TestIdWriteToReadFrom(t1 *testing.T) {
	t := ui.MakeT(t1)

	for _, value := range idWriteToReadFromData() {
		var id ObjectId
		t.AssertNoError(id.Set(value))

		var b bytes.Buffer

		_, err := id.WriteTo(&b)
		t.AssertNoError(err)

		var id2 ObjectId

		_, err = id2.ReadFrom(&b)
		t.AssertNoError(err)

		t.AssertEqual(id.String(), id2.String())
	}
}

// A quoted-literal value scans as a single TokenTypeLiteral token, but
// ScanExactlyOneSeqWithDotAllowedInIdenfierFromString still reports
// ErrMoreThanOneSeq (CanScan() is true post-literal, before EOF is
// observed) -- so Set's error-fallback path re-routes to SetBlob, which
// wraps the RAW string (quote characters included) in a single
// TokenTypeIdentifier token and unconditionally sets Genre to Blob,
// producing a Blob object-id literally named `"quoted value"`, quotes
// and all, rather than erroring or stripping the quotes. This is
// surprising but is the actual, tested contract: quoting has no special
// meaning to Set on a genre-less ObjectId, it's just characters in a
// blob name.
func TestSetOnQuotedStringProducesLiteralBlobName(t1 *testing.T) {
	t := ui.MakeT(t1)

	var id ObjectId
	t.AssertNoError(id.Set(`"quoted value"`))
	t.AssertEqual(genres.Blob, id.GetGenre())
	t.AssertEqualStrings(`"quoted value"`, id.String())
}
