package tridex

import (
	"slices"
	"sort"
	"testing"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

func TestMarshalBinaryRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)
	testCases := []struct {
		name     string
		elements []string
	}{
		{
			name:     "empty",
			elements: nil,
		},
		{
			name:     "single element",
			elements: []string{"hello"},
		},
		{
			name:     "multiple elements",
			elements: []string{"123456", "654321", "5"},
		},
		{
			name: "prefix overlapping",
			elements: []string{
				"12",
				"121",
				"127",
				"128",
				"123456",
				"654321",
			},
		},
		{
			name: "realistic tags",
			elements: []string{
				"person-john",
				"person-eric",
				"zz-archive",
				"zz-archive-recycle",
				"zz-archive-duplicate",
			},
		},
		{
			name: "degenerate prefix pair",
			elements: []string{
				"mew",
				"mewtwo",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(ui.MakeTestCaseInfo(tc.name), func(t *ui.T) {
			original := Make(tc.elements...)

			marshaler := original.(*Tridex)

			bs, err := marshaler.MarshalBinary()
			t.AssertNoError(err)

			restored := Make()
			unmarshaler := restored.(*Tridex)

			t.AssertNoError(unmarshaler.UnmarshalBinary(bs))

			expectedAll := slices.Collect(original.All())
			actualAll := slices.Collect(restored.All())

			sort.Strings(expectedAll)
			sort.Strings(actualAll)

			t.AssertEqual(expectedAll, actualAll)

			t.AssertEqual(original.Len(), restored.Len())

			for _, e := range tc.elements {
				t.AssertTrue(restored.ContainsExpansion(e), "restored tridex missing element "+e)

				expectedAbbr := original.Abbreviate(e)
				actualAbbr := restored.Abbreviate(e)

				t.AssertEqualStrings(expectedAbbr, actualAbbr)
			}
		})
	}
}

func TestMarshalBinaryDeterministic(t1 *testing.T) {
	t := ui.MakeT(t1)
	elements := []string{"zz-archive", "person-john", "todo", "priority-0_must"}

	first := Make(elements...)
	bs1, err := first.(*Tridex).MarshalBinary()
	t.AssertNoError(err)

	second := Make(elements...)
	bs2, err := second.(*Tridex).MarshalBinary()
	t.AssertNoError(err)

	t.AssertEqual(bs1, bs2)
}
