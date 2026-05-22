package ids

import (
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

func TestBlobExt(t1 *testing.T) {
	t := ui.MakeT(t1)
	var e TypeStruct

	err := e.Set("!md")
	t.AssertNoError(err)

	actual := e.StringSansOp()
	expected := "md"

	t.AssertEqualStrings(expected, actual)
}

func TestBlobExt1(t1 *testing.T) {
	t := ui.MakeT(t1)
	var e TypeStruct

	err := e.Set("md")
	t.AssertNoError(err)

	actual := e.StringSansOp()
	expected := "md"

	t.AssertEqualStrings(expected, actual)
}
