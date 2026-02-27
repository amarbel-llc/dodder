package ids

import (
	"fmt"
	"testing"
	tyme "time"

	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"code.linenisgreat.com/dodder/go/lib/echo/thyme"
)

func TestTaiSet(t1 *testing.T) {
	t := ui.T{T: t1}

	inSec := int64(2052235243)
	inASec := int64(336092000000000000)
	in := fmt.Sprintf("%d.%d", inSec, inASec)

	var sut Tai

	var err error

	if err = sut.Set(in); err != nil {
		t.Fatalf("failed to parse value into tai: %q. %s", in, err)
	}

	if err != nil {
		t.Fatalf("expected no error but got %s", err)
	}

	if sut.tai.Sec != inSec {
		t.Fatalf("expected Sec value '%d' but got '%d'", inSec, sut.tai.Sec)
	}

	if sut.tai.Asec != inASec {
		t.Fatalf("expected ASec value '%d' but got '%d'", inASec, sut.tai.Asec)
	}
}

func TestTaiSet2(t1 *testing.T) {
	t := ui.T{T: t1}

	inSec := int64(2052235243)
	inASec := int64(336092)
	inASecEx := int64(336092000000000000)
	in := fmt.Sprintf("%d.%d", inSec, inASec)

	var sut Tai

	var err error

	if err = sut.Set(in); err != nil {
		t.Fatalf("failed to parse value into tai: %q. %s", in, err)
	}

	if err != nil {
		t.Fatalf("expected no error but got %s", err)
	}

	if sut.tai.Sec != inSec {
		t.Fatalf("expected Sec value '%d' but got '%d'", inSec, sut.tai.Sec)
	}

	if sut.tai.Asec != inASecEx {
		t.Fatalf("expected ASec value '%d' but got '%d'", inASecEx, sut.tai.Asec)
	}
}

func TestTaiWithIndex(t1 *testing.T) {
	t := ui.T{T: t1}

	u := int64(1673549470)

	sut := TaiFromTimeWithIndex(
		thyme.Tyme(tyme.Unix(u, 0)),
		1,
	)

	if sut.tai.Sec != 2052240707 {
		t.Fatalf("expected Sec value '%d' but got '%d'", 2052240707, sut.tai.Sec)
	}

	if sut.tai.Asec != 1 {
		t.Fatalf("expected ASec value '%d' but got '%d'", 1, sut.tai.Asec)
	}

	ex := "2052240707.000000000000000001"
	if sut.String() != ex {
		t.Fatalf("expected .String() %q but got %q", ex, sut.String())
	}
}
