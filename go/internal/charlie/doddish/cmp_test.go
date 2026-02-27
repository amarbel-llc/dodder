package doddish

import (
	"testing"

	"code.linenisgreat.com/dodder/go/lib/alfa/cmp"
	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
)

type cmpTestCase struct {
	left, right string
	expected    cmp.Result
}

func getCmpTestCases() []ui.TestCase[cmpTestCase] {
	return []ui.TestCase[cmpTestCase]{
		ui.MakeTestCase(
			"same tag is equal",
			cmpTestCase{
				left:     "tag",
				right:    "tag",
				expected: cmp.Equal,
			},
		),
		ui.MakeTestCase(
			"tag with prefix is less",
			cmpTestCase{
				left:     "-tag",
				right:    "tag",
				expected: cmp.Less,
			},
		),
		ui.MakeTestCase(
			"zettel id is greater",
			cmpTestCase{
				left:     "zettel/id",
				right:    "tag",
				expected: cmp.Greater,
			},
		),
	}
}

func (testCase cmpTestCase) Test(t *ui.T) {
	left := makeSeqsFromString(t, testCase.left)
	right := makeSeqsFromString(t, testCase.right)
	actual := SeqsCompare(left, right)
	t.AssertEqual(testCase.expected, actual)
}

func TestCmp(t1 *testing.T) {
	t := ui.T{T: t1}

	for _, testCase := range getCmpTestCases() {
		t.Run(
			testCase,
			func(t *ui.T) {
				testCase.GetBlob().Test(t)
			},
		)
	}
}
