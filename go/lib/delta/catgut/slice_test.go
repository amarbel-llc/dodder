package catgut

import (
	"bytes"
	"reflect"
	"testing"

	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

type testCaseOverlap struct {
	Slice
	ExpectedOverlap               []byte
	ExpectedFirst, ExpectedSecond int
}

func getTestCasesOverlap() []testCaseOverlap {
	return []testCaseOverlap{
		{
			Slice: Slice{
				data: [2][]byte{
					[]byte("test"),
					[]byte("string"),
				},
			},
			ExpectedOverlap: []byte{'e', 's', 't', 's', 't', 'r'},
			ExpectedFirst:   3,
			ExpectedSecond:  3,
		},
		{
			Slice: Slice{
				data: [2][]byte{
					[]byte("te"),
					[]byte("ststring"),
				},
			},
			ExpectedOverlap: []byte{'t', 'e', 's', 't', 's'},
			ExpectedFirst:   2,
			ExpectedSecond:  3,
		},
		{
			Slice: Slice{
				data: [2][]byte{
					[]byte("t"),
					[]byte("eststring"),
				},
			},
			ExpectedOverlap: []byte{'t', 'e', 's', 't'},
			ExpectedFirst:   1,
			ExpectedSecond:  3,
		},
		{
			Slice: Slice{
				data: [2][]byte{
					[]byte("test"),
					[]byte("st"),
				},
			},
			ExpectedOverlap: []byte{'e', 's', 't', 's', 't'},
			ExpectedFirst:   3,
			ExpectedSecond:  2,
		},
		{
			Slice: Slice{
				data: [2][]byte{
					[]byte("te"),
					[]byte("st"),
				},
			},
			ExpectedOverlap: []byte{'t', 'e', 's', 't'},
			ExpectedFirst:   2,
			ExpectedSecond:  2,
		},
	}
}

func TestSliceOverlap(t1 *testing.T) {
	t := ui.T{T: t1}

	for _, tc := range getTestCasesOverlap() {
		sut := tc.Slice

		overlap, first, second := sut.Overlap()

		if first != tc.ExpectedFirst {
			t.AssertEqual(tc.ExpectedFirst, first)
		}

		if second != tc.ExpectedSecond {
			t.AssertEqual(tc.ExpectedSecond, second)
		}

		actual := overlap[:first+second]

		if !reflect.DeepEqual(actual, tc.ExpectedOverlap) {
			t.AssertEqual(tc.ExpectedOverlap, actual)
		}
	}
}

type testCaseSlice struct {
	Slice
	ExpectedSliceData           []byte
	ExpectedLeft, ExpectedRight []byte
	Left, Right                 int
}

func getTestCasesSlice() []testCaseSlice {
	return []testCaseSlice{
		{
			Slice: Slice{
				data: [2][]byte{
					[]byte("te"),
					[]byte("st"),
				},
			},
			ExpectedSliceData: []byte{'t', 'e', 's', 't'},
			Left:              0,
			Right:             4,
		},
		{
			Slice: Slice{
				data: [2][]byte{
					[]byte("te"),
					[]byte("st"),
				},
			},
			ExpectedSliceData: []byte{'e', 's', 't'},
			Left:              1,
			Right:             4,
		},
		{
			Slice: Slice{
				data: [2][]byte{
					[]byte("te"),
					[]byte("st"),
				},
			},
			ExpectedSliceData: []byte{'s', 't'},
			Left:              2,
			Right:             4,
		},
		{
			Slice: Slice{
				data: [2][]byte{
					[]byte("te"),
					[]byte("st"),
				},
			},
			ExpectedSliceData: []byte{'t'},
			Left:              3,
			Right:             4,
		},
		{
			Slice: Slice{
				data: [2][]byte{
					[]byte("te"),
					[]byte("st"),
				},
			},
			ExpectedSliceData: []byte{'t', 'e'},
			Left:              0,
			Right:             2,
		},
		{
			Slice: Slice{
				data: [2][]byte{
					[]byte("test"),
					[]byte(""),
				},
			},
			ExpectedSliceData: []byte{'t', 'e', 's', 't'},
			Left:              0,
			Right:             4,
		},
	}
}

func TestSliceSlice(t1 *testing.T) {
	t := ui.T{T: t1}

	for _, tc := range getTestCasesSlice() {
		sut := tc.Slice

		sub := sut.Slice(tc.Left, tc.Right)
		actual := sub.Bytes()

		if !bytes.Equal(tc.ExpectedSliceData, actual) {
			t.AssertEqual(tc.ExpectedSliceData, actual)
		}
	}
}
