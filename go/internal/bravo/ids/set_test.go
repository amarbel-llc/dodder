package ids

import (
	"testing"

	"code.linenisgreat.com/dodder/go/lib/0/collections_slice"
	"code.linenisgreat.com/dodder/go/lib/alfa/expansion"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter_set"
	"code.linenisgreat.com/dodder/go/lib/bravo/collections_delta"
	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

func TestExpandedRight(t1 *testing.T) {
	t := ui.MakeT(t1)
	tags := MakeTagSetFromSlice(
		MustTag("project-2021-dodder"),
		MustTag("zz-archive-task-done"),
	)

	ex := collections_slice.Collect(expansion.ExpandMany(tags.All(), expansion.ExpanderRight))

	expected := []string{
		"project",
		"project-2021",
		"project-2021-dodder",
		"zz",
		"zz-archive",
		"zz-archive-task",
		"zz-archive-task-done",
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

func TestPrefixIntersection(t1 *testing.T) {
	t := ui.MakeT(t1)
	s := MakeTagSetFromSlice(
		MustTag("project-2021-dodder"),
		MustTag("zz-archive-task-done"),
	)

	ex := IntersectPrefixes(s, MustTag("project"))

	expected := []string{
		"project-2021-dodder",
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

// func TestExpansionRight(t *testing.T) {
// 	e := MustTag("this-is-a-tag")
// 	ex := e.Expanded(ExpanderRight{})
// 	expected := []string{
// 		"this",
// 		"this-is",
// 		"this-is-a",
// 		"this-is-a-tag",
// 	}

// 	actual := ex.SortedString()

// 	if !stringSliceEquals(actual, expected) {
// 		t.Errorf(
// 			"expanded tags don't match:\nexpected: %q\n  actual: %q",
// 			expected,
// 			actual,
// 		)
// 	}
// }

func TestDelta1(t1 *testing.T) {
	t := ui.MakeT(t1)
	from := MakeTagSetFromSlice(
		MustTag("project-2021-dodder"),
		MustTag("task-todo"),
	)

	to := MakeTagSetFromSlice(
		MustTag("project-2021-dodder"),
		MustTag("zz-archive-task-done"),
	)

	d := collections_delta.MakeSetDelta(from, to)

	c_expected := MakeTagSetFromSlice(
		MustTag("zz-archive-task-done"),
	)

	if !quiter_set.Equals(c_expected, d.GetAdded()) {
		t.Errorf("expected\n%s\nactual:\n%s", c_expected, d.GetAdded())
	}

	d_expected := MakeTagSetFromSlice(
		MustTag("task-todo"),
	)

	if !quiter_set.Equals(d_expected, d.GetRemoved()) {
		t.Errorf("expected\n%s\nactual:\n%s", d_expected, d.GetRemoved())
	}
}
