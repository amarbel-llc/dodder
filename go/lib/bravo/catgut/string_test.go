package catgut

import (
	"fmt"
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/alfa/cmp"
	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
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
	t := ui.MakeT(t1)
	for _, tc := range getTestCasesCompare() {
		t.Run(
			ui.MakeTestCaseInfo(fmt.Sprintf("%#v", tc)),
			func(t *ui.T) {
				a, _ := MakeFromString(tc.a) //repool:owned
				b, _ := MakeFromString(tc.b) //repool:owned

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
	t := ui.MakeT(t1)
	for _, tc := range getTestCasesComparePartial() {
		t.Run(
			ui.MakeTestCaseInfo(fmt.Sprintf("%#v", tc)),
			func(t *ui.T) {
				a, _ := MakeFromString(tc.a) //repool:owned
				b, _ := MakeFromString(tc.b) //repool:owned

				actual := a.ComparePartial(b)

				if actual != tc.expected {
					t.Errorf("expected %d but got %d", tc.expected, actual)
				}
			},
		)
	}
}
