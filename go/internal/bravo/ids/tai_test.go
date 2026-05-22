package ids

import (
	"fmt"
	"testing"
	tyme "time"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/thyme"
)

func TestTaiSet(t1 *testing.T) {
	t := ui.MakeT(t1)

	inSec := int64(2052235243)
	inASec := int64(336092000000000000)
	in := fmt.Sprintf("%d.%d", inSec, inASec)

	var sut Tai

	err := sut.Set(in)
	t.AssertNoError(err)

	t.AssertEqual(inSec, sut.tai.Sec)
	t.AssertEqual(inASec, sut.tai.Asec)
}

func TestTaiSet2(t1 *testing.T) {
	t := ui.MakeT(t1)

	inSec := int64(2052235243)
	inASec := int64(336092)
	inASecEx := int64(336092000000000000)
	in := fmt.Sprintf("%d.%d", inSec, inASec)

	var sut Tai

	err := sut.Set(in)
	t.AssertNoError(err)

	t.AssertEqual(inSec, sut.tai.Sec)
	t.AssertEqual(inASecEx, sut.tai.Asec)
}

func TestTaiWithIndex(t1 *testing.T) {
	t := ui.MakeT(t1)

	u := int64(1673549470)

	sut := TaiFromTimeWithIndex(
		thyme.Tyme(tyme.Unix(u, 0)),
		1,
	)

	t.AssertEqual(int64(2052240707), sut.tai.Sec)
	t.AssertEqual(int64(1), sut.tai.Asec)

	ex := "2052240707.000000000000000001"
	t.AssertEqualStrings(ex, sut.String())
}
