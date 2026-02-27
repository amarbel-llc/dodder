package ids

import (
	"testing"

	"code.linenisgreat.com/dodder/go/lib/alfa/quiter_collection"
	"code.linenisgreat.com/dodder/go/lib/bravo/collections_slice"
	"code.linenisgreat.com/dodder/go/lib/bravo/quiter"
	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
	"code.linenisgreat.com/dodder/go/lib/charlie/expansion"
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
	t := ui.T{T: t1}

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

	if stringSliceEquals(expected, actual) {
		t.Errorf("expected unequal slices")
	}
}

func TestStringSliceEquals(t1 *testing.T) {
	t := ui.T{T: t1}

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

	if !stringSliceEquals(expected, actual) {
		t.Errorf("expected equal slices")
	}
}

func TestExpansionAll(t1 *testing.T) {
	t := ui.T{T: t1}
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
	t := ui.T{T: t1}

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
	t := ui.T{T: t1}
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
