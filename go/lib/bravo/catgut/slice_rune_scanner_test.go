package catgut

import (
	"testing"
	"unicode/utf8"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
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
	t := ui.MakeT(t1)
	for _, expected := range testSliceRuneScannerDataValid() {
		t.Run(
			ui.MakeTestCaseInfo(expected.String()),
			func(t *ui.T) {
				sut := MakeSliceRuneScanner(expected)

				for _, rEx := range expected.String() {
					widthEx := utf8.RuneLen(rEx)
					r, width, ok := sut.Scan()

					t.AssertTrue(ok, "expected successful scan")

					t.AssertEqual(rEx, r)

					t.AssertEqual(widthEx, width)

					t.AssertNoError(sut.UnreadRune())
					sut.ReadRune()
				}

				_, _, ok := sut.Scan()

				t.AssertFalse(ok, "expected unsuccessful scan")
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
	t := ui.MakeT(t1)
	for _, expected := range testSliceRuneScannerDataInvalid() {
		t.Run(
			ui.MakeTestCaseInfo(expected.String()),
			func(t *ui.T) {
				sut := MakeSliceRuneScanner(expected)

				_, _, ok := sut.Scan()

				t.AssertFalse(ok, "expected unsuccessful scan")
			},
		)
	}
}
