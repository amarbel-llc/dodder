package ids

import (
	"testing"

	"code.linenisgreat.com/dodder/go/lib/0/collections_slice"
	"code.linenisgreat.com/dodder/go/lib/alfa/expansion"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter_collection"
	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

func stringSliceEquals(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestStringSliceUnequal(t1 *testing.T) {
	t := ui.MakeT(t1)

	expected := []string{
		"this",
		"is",
		"a",
	}

	actual := []string{
		"this",
		"is",
		"a",
		"tag",
	}

	t.AssertFalse(stringSliceEquals(expected, actual), "expected unequal slices")
}

func TestStringSliceEquals(t1 *testing.T) {
	t := ui.MakeT(t1)

	expected := []string{
		"this",
		"is",
		"a",
		"tag",
	}

	actual := []string{
		"this",
		"is",
		"a",
		"tag",
	}

	t.AssertTrue(stringSliceEquals(expected, actual), "expected equal slices")
}

func TestExpansionAll(t1 *testing.T) {
	t := ui.MakeT(t1)
	e := MustTag("this-is-a-tag")

	ex := expansion.ExpandIntoSlice[TagStruct](
		e.String(),
		expansion.ExpanderAll,
	)

	expected := []string{
		"a",
		"a-tag",
		"is",
		"is-a-tag",
		"tag",
		"this",
		"this-is",
		"this-is-a",
		"this-is-a-tag",
	}

	actual := quiter.SortedStrings(ex)

	if !stringSliceEquals(actual, expected) {
		t.Errorf(
			"expanded tags don't match:\nexpected: %q\n  actual: %q",
			expected,
			actual,
		)
	}
}

func TestExpansionRight(t1 *testing.T) {
	t := ui.MakeT(t1)

	e := MustTag("this-is-a-tag")

	ex := expansion.ExpandIntoSlice[TagStruct](
		e.String(),
		expansion.ExpanderRight,
	)

	expected := []string{
		"this",
		"this-is",
		"this-is-a",
		"this-is-a-tag",
	}

	actual := quiter.SortedStrings(ex)

	if !stringSliceEquals(actual, expected) {
		t.Errorf(
			"expanded tags don't match:\nexpected: %q\n  actual: %q",
			expected,
			actual,
		)
	}
}

func TestExpansionRightTypeNone(t1 *testing.T) {
	t := ui.MakeT(t1)
	tipe := MustTypeStruct("md")

	actual := expansion.ExpandIntoSlice[TypeStruct](
		tipe.String(),
		expansion.ExpanderRight,
	)

	expected := collections_slice.Slice[TypeStruct]{
		MustTypeStruct("md"),
	}

	if !quiter_collection.Equals(actual, expected) {
		t.Errorf(
			"expanded tags don't match:\nexpected: %q\n  actual: %q",
			expected,
			actual,
		)
	}
}
