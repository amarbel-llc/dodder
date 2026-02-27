package catgut

import (
	"testing"
	"unicode/utf8"

	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

func testSliceRuneScannerDataValid() []Slice {
	return []Slice{
		{
			data: [2][]byte{[]byte("string")},
		},
		{
			data: [2][]byte{[]byte("\u2318")},
		},
		{
			data: [2][]byte{
				[]byte("xxx\xe2\x8c"),
				[]byte("\x98"),
			},
		},
		{
			data: [2][]byte{
				[]byte("123456"),
				[]byte("abcdef"),
			},
		},
		{
			data: [2][]byte{
				[]byte("123"),
				[]byte("456"),
			},
		},
	}
}

func TestSliceRuneScannerValid(t1 *testing.T) {
	for _, expected := range testSliceRuneScannerDataValid() {
		t1.Run(
			expected.String(),
			func(t2 *testing.T) {
				t := ui.T{T: t2}

				sut := MakeSliceRuneScanner(expected)

				for _, rEx := range expected.String() {
					widthEx := utf8.RuneLen(rEx)
					r, width, ok := sut.Scan()

					if !ok {
						t.Errorf("expected successful scan")
					}

					if r != rEx {
						t.Errorf("expected %c but got %c", rEx, r)
					}

					if width != widthEx {
						t.Errorf("expected %d but got %d", widthEx, width)
					}

					t.AssertNoError(sut.UnreadRune())
					sut.ReadRune()
				}

				_, _, ok := sut.Scan()

				if ok {
					t.Errorf("expected unsuccessful scan")
				}
			},
		)
	}
}

func testSliceRuneScannerDataInvalid() []Slice {
	return []Slice{
		{
			data: [2][]byte{[]byte("\xbd\xb2\x3d\xbc\x20\xe2\x8c\x98")},
		},
	}
}

func TestSliceRuneScannerInvalid(t1 *testing.T) {
	for _, expected := range testSliceRuneScannerDataInvalid() {
		t1.Run(
			expected.String(),
			func(t2 *testing.T) {
				t := ui.T{T: t2}

				sut := MakeSliceRuneScanner(expected)

				_, _, ok := sut.Scan()

				if ok {
					t.Errorf("expected unsuccessful scan")
				}
			},
		)
	}
}
