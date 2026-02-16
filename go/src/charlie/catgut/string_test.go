package catgut

import (
	"fmt"
	"testing"

	"code.linenisgreat.com/dodder/go/src/alfa/cmp"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
)

func TestMain(m *testing.M) {
	ui.SetTesting()
	m.Run()
}

type testCaseCompare struct {
	a, b     string
	expected cmp.Result
}

func getTestCasesCompare() []testCaseCompare {
	return []testCaseCompare{
		{
			a:        "test",
			b:        "test",
			expected: cmp.Equal,
		},
		{
			a:        "xest",
			b:        "test",
			expected: cmp.Greater,
		},
		{
			a:        "",
			b:        "test",
			expected: cmp.Less,
		},
	}
}

func TestCompare(t1 *testing.T) {
	for _, tc := range getTestCasesCompare() {
		t1.Run(
			fmt.Sprintf("%#v", tc),
			func(t1 *testing.T) {
				t := ui.T{T: t1}

				a, _ := MakeFromString(tc.a)
				b, _ := MakeFromString(tc.b)

				actual := a.Compare(b)

				if actual != tc.expected {
					t.Errorf("expected %d but got %d", tc.expected, actual)
				}
			},
		)
	}
}

func getTestCasesComparePartial() []testCaseCompare {
	return []testCaseCompare{
		{
			a:        "test",
			b:        "test",
			expected: cmp.Equal,
		},
		{
			a:        "tests",
			b:        "test",
			expected: cmp.Equal,
		},
		{
			a:        "test",
			b:        "tests",
			expected: cmp.Less,
		},
		{
			a:        "",
			b:        "test",
			expected: cmp.Less,
		},
	}
}

func TestComparePartial(t1 *testing.T) {
	for _, tc := range getTestCasesComparePartial() {
		t1.Run(
			fmt.Sprintf("%#v", tc),
			func(t1 *testing.T) {
				t := ui.T{T: t1}

				a, _ := MakeFromString(tc.a)
				b, _ := MakeFromString(tc.b)

				actual := a.ComparePartial(b)

				if actual != tc.expected {
					t.Errorf("expected %d but got %d", tc.expected, actual)
				}
			},
		)
	}
}
