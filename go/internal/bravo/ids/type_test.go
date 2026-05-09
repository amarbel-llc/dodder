package ids

import (
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

func TestBlobExt(t1 *testing.T) {
	t := ui.MakeT(t1)
	var e TypeStruct
	var err error

	if err = e.Set("!md"); err != nil {
		t.Fatalf("%s", err)
	}

	actual := e.StringSansOp()
	expected := "md"

	if expected != actual {
		t.Fatalf("expected %q, but got %q", expected, actual)
	}
}

func TestBlobExt1(t1 *testing.T) {
	t := ui.MakeT(t1)
	var e TypeStruct
	var err error

	if err = e.Set("md"); err != nil {
		t.Fatalf("%s", err)
	}

	actual := e.StringSansOp()
	expected := "md"

	if expected != actual {
		t.Fatalf("expected %q, but got %q", expected, actual)
	}
}
