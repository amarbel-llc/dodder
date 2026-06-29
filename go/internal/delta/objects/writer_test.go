package objects

import (
	"strings"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/0/hyphence"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

func TestWriter1(t1 *testing.T) {
	t := ui.MakeT(t1)

	expectedOut := `---
metadatei
---

blob
`

	out := &strings.Builder{}

	sut := hyphence.Writer{
		Metadata: strings.NewReader("metadatei\n"),
		Blob:     strings.NewReader("blob\n"),
	}

	sut.WriteTo(out)

	t.AssertEqualStrings(expectedOut, out.String())
}

func TestWriter2(t1 *testing.T) {
	t := ui.MakeT(t1)

	expectedOut := `---
metadatei
---
`

	out := &strings.Builder{}

	sut := hyphence.Writer{
		Metadata: strings.NewReader("metadatei\n"),
	}

	sut.WriteTo(out)

	t.AssertEqualStrings(expectedOut, out.String())
}

func TestWriter3(t1 *testing.T) {
	t := ui.MakeT(t1)

	expectedOut := `blob
`

	out := &strings.Builder{}

	sut := hyphence.Writer{
		Blob: strings.NewReader("blob\n"),
	}

	sut.WriteTo(out)

	t.AssertEqualStrings(expectedOut, out.String())
}
