package tridex

import (
	"reflect"
	"slices"
	"sort"
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
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
			if err != nil {
				t.Fatalf("MarshalBinary failed: %s", err)
			}

			restored := Make()
			unmarshaler := restored.(*Tridex)

			if err := unmarshaler.UnmarshalBinary(bs); err != nil {
				t.Fatalf("UnmarshalBinary failed: %s", err)
			}

			expectedAll := slices.Collect(original.All())
			actualAll := slices.Collect(restored.All())

			sort.Strings(expectedAll)
			sort.Strings(actualAll)

			if !reflect.DeepEqual(expectedAll, actualAll) {
				t.Errorf(
					"round-trip mismatch:\n  expected: %v\n  got:      %v",
					expectedAll,
					actualAll,
				)
			}

			if original.Len() != restored.Len() {
				t.Errorf(
					"Len mismatch: expected %d, got %d",
					original.Len(),
					restored.Len(),
				)
			}

			for _, e := range tc.elements {
				if !restored.ContainsExpansion(e) {
					t.Errorf("restored tridex missing element %q", e)
				}

				expectedAbbr := original.Abbreviate(e)
				actualAbbr := restored.Abbreviate(e)

				if expectedAbbr != actualAbbr {
					t.Errorf(
						"abbreviation mismatch for %q: expected %q, got %q",
						e,
						expectedAbbr,
						actualAbbr,
					)
				}
			}
		})
	}
}

func TestMarshalBinaryDeterministic(t1 *testing.T) {
	t := ui.MakeT(t1)
	elements := []string{"zz-archive", "person-john", "todo", "priority-0_must"}

	first := Make(elements...)
	bs1, err := first.(*Tridex).MarshalBinary()
	if err != nil {
		t.Fatalf("first MarshalBinary failed: %s", err)
	}

	second := Make(elements...)
	bs2, err := second.(*Tridex).MarshalBinary()
	if err != nil {
		t.Fatalf("second MarshalBinary failed: %s", err)
	}

	if !reflect.DeepEqual(bs1, bs2) {
		t.Errorf("MarshalBinary not deterministic: different bytes for same content")
	}
}
