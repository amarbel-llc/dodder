package heap

import (
	"reflect"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/bravo/ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/values"
)

func TestReset(t1 *testing.T) {
	t := ui.T{T: t1}

	els := []*values.Int{
		values.MakeInt(1),
		values.MakeInt(0),
		values.MakeInt(3),
		values.MakeInt(4),
		values.MakeInt(2),
	}

	sut := MakeNewHeapFromSliceUnsorted[values.Int, *values.Int](
		makeTestFuncCmp(t),
		values.IntResetter{},
		els,
	)

	if sut.Len() != 5 {
		t.Fatalf("expected len 5 but got %d", sut.Len())
	}

	sut.Reset()

	if sut.Len() != 0 {
		t.Fatalf("expected len 0 but got %d", sut.Len())
	}
}

func TestSaveAndRestore(t1 *testing.T) {
	t := ui.T{T: t1}

	els := []*values.Int{
		values.MakeInt(1),
		values.MakeInt(0),
		values.MakeInt(3),
		values.MakeInt(4),
		values.MakeInt(2),
	}

	eql := values.IntEqualer{}

	sut := MakeNewHeapFromSliceUnsorted[values.Int, *values.Int](
		makeTestFuncCmp(t),
		values.IntResetter{},
		els,
	)

	checkAllElements := func() {
		defer sut.restore()
		for i := 0; i < 5; i++ {
			el, ok := sut.popAndSave()
			ex := values.MakeInt(i)

			if !ok {
				t.Fatalf(
					"expected pop and save to return an element but got nothing",
				)
			}

			if !eql.EqualsPtr(el, ex) {
				t.Fatalf(
					"expected pop and save to return %s but got %s",
					ex,
					el,
				)
			}
		}
	}

	checkAllElements()

	for i := 0; i < 5; i++ {
		el, ok := sut.Pop()
		ex := values.MakeInt(i)

		if !ok {
			t.Fatalf(
				"expected pop and save to return an element but got nothing. Idx: %d",
				i,
			)
		}

		if !eql.EqualsPtr(el, ex) {
			t.Fatalf("expected pop and save to return %s but got %s", ex, el)
		}
	}
}

func Test3Sorted(t1 *testing.T) {
	t := ui.T{T: t1}

	els := []*values.Int{
		values.MakeInt(1),
		values.MakeInt(0),
		values.MakeInt(3),
		values.MakeInt(4),
		values.MakeInt(2),
	}

	sut := MakeNewHeapFromSliceUnsorted[values.Int, *values.Int](
		makeTestFuncCmp(t),
		values.IntResetter{},
		els,
	)

	sorted := sut.Sorted()

	expected := []*values.Int{
		values.MakeInt(0),
		values.MakeInt(1),
		values.MakeInt(2),
		values.MakeInt(3),
		values.MakeInt(4),
	}

	if !reflect.DeepEqual([]*values.Int(sorted), expected) {
		t.Fatalf("expected %#v but got %#v", expected, sorted)
	}
}

func TestDupes(t1 *testing.T) {
	t := ui.T{T: t1}
	t.SkipTest()

	els := []*values.Int{
		values.MakeInt(1),
		values.MakeInt(0),
		values.MakeInt(3),
		values.MakeInt(3),
		values.MakeInt(4),
		values.MakeInt(2),
	}

	sut := MakeNewHeapFromSliceUnsorted[values.Int, *values.Int](
		makeTestFuncCmp(t),
		values.IntResetter{},
		els,
	)

	sorted := sut.Sorted()

	expected := []*values.Int{
		values.MakeInt(0),
		values.MakeInt(1),
		values.MakeInt(2),
		values.MakeInt(3),
		values.MakeInt(4),
	}

	if !reflect.DeepEqual([]*values.Int(sorted), expected) {
		t.Fatalf("expected %s but got %s", expected, sorted)
	}
}
