package ids

import (
	"testing"

	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
)

func TestMake(t1 *testing.T) {
	t := ui.T{T: t1}
	in := "ceroplastes/midtown"
	var sut ZettelId

	if err := sut.Set(in); err != nil {
		t.Errorf("expected no error but got: '%s'", err)
	}

	ex := in
	ac := sut.String()

	if ex != ac {
		t.Errorf("expected %q but got %q", ex, ac)
	}
}

func TestMakeHeadAndTail(t1 *testing.T) {
	t := ui.T{T: t1}
	k := "ceroplastes"
	s := "midtown"

	var sut *ZettelId
	var err error

	if sut, err = MakeZettelIdFromHeadAndTail(k, s); err != nil {
		t.Errorf("expected no error but got: '%s'", err)
	}

	ex := k + "/" + s
	ac := sut.String()

	if ex != ac {
		t.Errorf("expected %q but got %q", ex, ac)
	}
}
