package objects

import (
	"bytes"
	"testing"

	"github.com/amarbel-llc/madder/go/pkgs/hyphence"
	"github.com/amarbel-llc/purse-first/libs/dewey/alfa/pool"
	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

func Test1(t1 *testing.T) {
	t := ui.MakeT(t1)

	in := `---
metadatei
---

body
`
	mExpected := "metadatei\n"
	bExpected := "body\n"
	nExpected := int64(len(in))

	mr := &bytes.Buffer{}
	ar := &bytes.Buffer{}

	r := hyphence.Reader{
		Metadata: mr,
		Blob:     ar,
	}

	var n int64
	var err error

	reader, repool := pool.GetStringReader(in)
	defer repool()
	n, err = r.ReadFrom(reader)

	t.AssertEqual(nExpected, n)

	if err != nil {
		t.Errorf("expected no error but got %s", err)
	}

	t.AssertEqualStrings(mExpected, string(mr.Bytes()))

	t.AssertEqualStrings(bExpected, string(ar.Bytes()))
}

func Test2(t1 *testing.T) {
	t := ui.MakeT(t1)

	in := `---
metadatei
---
`
	mExpected := "metadatei\n"
	bExpected := ""
	nExpected := int64(len(in))

	mr := &bytes.Buffer{}
	ar := &bytes.Buffer{}

	r := hyphence.Reader{
		Metadata: mr,
		Blob:     ar,
	}

	var n int64
	var err error

	reader, repool := pool.GetStringReader(in)
	defer repool()
	n, err = r.ReadFrom(reader)

	t.AssertEqual(nExpected, n)

	if err != nil {
		t.Errorf("expected no error but got %s", err)
	}

	t.AssertEqualStrings(mExpected, string(mr.Bytes()))

	t.AssertEqualStrings(bExpected, string(ar.Bytes()))
}
