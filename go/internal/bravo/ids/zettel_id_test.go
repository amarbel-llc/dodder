package ids

import (
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

func TestMake(t1 *testing.T) {
	t := ui.MakeT(t1)
	in := "ceroplastes/midtown"
	var sut ZettelId

	if err := sut.Set(in); err != nil {
		t.Errorf("expected no error but got: '%s'", err)
	}

	ex := in
	ac := sut.String()

	t.AssertEqualStrings(ex, ac)
}

func TestMakeHeadAndTail(t1 *testing.T) {
	t := ui.MakeT(t1)
	k := "ceroplastes"
	s := "midtown"

	var sut *ZettelId
	var err error

	if sut, err = MakeZettelIdFromHeadAndTail(k, s); err != nil {
		t.Errorf("expected no error but got: '%s'", err)
	}

	ex := k + "/" + s
	ac := sut.String()

	t.AssertEqualStrings(ex, ac)
}
