package coordinates

import "testing"

func TestEquals(t *testing.T) {
	p := ZettelIdCoordinate{Left: 1, Right: 1}
	p1 := ZettelIdCoordinate{Left: 1, Right: 1}

	if !p.Equals(p1) {
		t.Errorf("expected equality")
	}
}

func TestNotEquals(t *testing.T) {
	p := ZettelIdCoordinate{Left: 1, Right: 1}
	p1 := ZettelIdCoordinate{Left: 0, Right: 1}

	if p.Equals(p1) {
		t.Errorf("expected inequality")
	}
}

func TestToId1(t *testing.T) {
	assertToId(
		t,
		ZettelIdCoordinate{Left: 0, Right: 0},
		1,
	)
}

func TestToId2(t *testing.T) {
	assertToId(
		t,
		ZettelIdCoordinate{Left: 0, Right: 1},
		2,
	)
}

func TestToId42(t *testing.T) {
	assertToId(
		t,
		ZettelIdCoordinate{Left: 5, Right: 3},
		42,
	)
}

func TestFromId5(t *testing.T) {
	assertFromId(t, "5", ZettelIdCoordinate{Left: 1, Right: 1})
}

func TestFromId745(t *testing.T) {
	assertFromId(t, "745", ZettelIdCoordinate{Left: 3, Right: 35})
}

func TestFromId10469(t *testing.T) {
	assertFromId(t, "10469", ZettelIdCoordinate{Left: 28, Right: 116})
}

func TestFromId1(t *testing.T) {
	assertFromId(t, "1", ZettelIdCoordinate{Left: 0, Right: 0})
}

func TestFromId2(t *testing.T) {
	assertFromId(t, "2", ZettelIdCoordinate{Left: 0, Right: 1})
}

func TestFromId3(t *testing.T) {
	assertFromId(t, "3", ZettelIdCoordinate{Left: 1, Right: 0})
}

func TestFromId42(t *testing.T) {
	assertFromId(t, "42", ZettelIdCoordinate{Left: 5, Right: 3})
}

func TestFromId567(t *testing.T) {
	assertFromId(t, "567", ZettelIdCoordinate{Left: 5, Right: 28})
}

func TestFromId672(t *testing.T) {
	assertFromId(t, "672", ZettelIdCoordinate{Left: 5, Right: 31})
}

func assertFromId(t *testing.T, n string, expected ZettelIdCoordinate) {
	t.Helper()

	p := &ZettelIdCoordinate{}
	p.Set(n)

	if !p.Equals(expected) {
		t.Errorf("expected %v but got %v", expected, p)
	}
}

func assertToId(t *testing.T, p ZettelIdCoordinate, expected Int) {
	t.Helper()

	id := p.Id()

	if id != expected {
		t.Errorf("expected %d but got %d", expected, id)
	}
}
