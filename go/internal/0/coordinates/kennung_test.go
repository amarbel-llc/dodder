package coordinates

import (
	"testing"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

func TestEquals(t1 *testing.T) {
	t := ui.MakeT(t1)
	p := ZettelIdCoordinate{Left: 1, Right: 1}
	p1 := ZettelIdCoordinate{Left: 1, Right: 1}

	t.AssertTrue(p.Equals(p1), "expected equality")
}

func TestNotEquals(t1 *testing.T) {
	t := ui.MakeT(t1)
	p := ZettelIdCoordinate{Left: 1, Right: 1}
	p1 := ZettelIdCoordinate{Left: 0, Right: 1}

	t.AssertFalse(p.Equals(p1), "expected inequality")
}

func TestToId1(t1 *testing.T) {
	t := ui.MakeT(t1)
	assertToId(
		&t,
		ZettelIdCoordinate{Left: 0, Right: 0},
		1,
	)
}

func TestToId2(t1 *testing.T) {
	t := ui.MakeT(t1)
	assertToId(
		&t,
		ZettelIdCoordinate{Left: 0, Right: 1},
		2,
	)
}

func TestToId42(t1 *testing.T) {
	t := ui.MakeT(t1)
	assertToId(
		&t,
		ZettelIdCoordinate{Left: 5, Right: 3},
		42,
	)
}

func TestFromId5(t1 *testing.T) {
	t := ui.MakeT(t1)
	assertFromId(&t, "5", ZettelIdCoordinate{Left: 1, Right: 1})
}

func TestFromId745(t1 *testing.T) {
	t := ui.MakeT(t1)
	assertFromId(&t, "745", ZettelIdCoordinate{Left: 3, Right: 35})
}

func TestFromId10469(t1 *testing.T) {
	t := ui.MakeT(t1)
	assertFromId(&t, "10469", ZettelIdCoordinate{Left: 28, Right: 116})
}

func TestFromId1(t1 *testing.T) {
	t := ui.MakeT(t1)
	assertFromId(&t, "1", ZettelIdCoordinate{Left: 0, Right: 0})
}

func TestFromId2(t1 *testing.T) {
	t := ui.MakeT(t1)
	assertFromId(&t, "2", ZettelIdCoordinate{Left: 0, Right: 1})
}

func TestFromId3(t1 *testing.T) {
	t := ui.MakeT(t1)
	assertFromId(&t, "3", ZettelIdCoordinate{Left: 1, Right: 0})
}

func TestFromId42(t1 *testing.T) {
	t := ui.MakeT(t1)
	assertFromId(&t, "42", ZettelIdCoordinate{Left: 5, Right: 3})
}

func TestFromId567(t1 *testing.T) {
	t := ui.MakeT(t1)
	assertFromId(&t, "567", ZettelIdCoordinate{Left: 5, Right: 28})
}

func TestFromId672(t1 *testing.T) {
	t := ui.MakeT(t1)
	assertFromId(&t, "672", ZettelIdCoordinate{Left: 5, Right: 31})
}

func assertFromId(t *ui.T, n string, expected ZettelIdCoordinate) {
	t.Helper()

	p := &ZettelIdCoordinate{}
	p.Set(n)

	t.AssertEqual(expected, *p)
}

func assertToId(t *ui.T, p ZettelIdCoordinate, expected Int) {
	t.Helper()

	id := p.Id()

	t.AssertEqual(expected, id)
}
