package quiter

import (
	"slices"
	"sort"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/0/collections_slice"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/cmp"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

// Deprecated: use collections_slice.SortedValuesBy
func SortedValuesBy[ELEMENT any](
	set interfaces.Collection[ELEMENT],
	cmp cmp.Func[ELEMENT],
) []ELEMENT {
	return collections_slice.SortedValuesBy(set, cmp)
}

// Deprecated: use collections_slice.SortedValues
func SortedValues[ELEMENT interfaces.Value](
	seq interfaces.Seq[ELEMENT],
) []ELEMENT {
	return collections_slice.SortedValues(seq)
}

func Strings[ELEMENT interfaces.Stringer](
	collections interfaces.Seq[interfaces.Collection[ELEMENT]],
) interfaces.Seq[string] {
	return func(yield func(string) bool) {
		for collection := range collections {
			if collection == nil {
				continue
			}

			for element := range collection.All() {
				if !yield(element.String()) {
					return
				}
			}
		}
	}
}

func SortedStrings[ELEMENT interfaces.Stringer](
	collections ...interfaces.Collection[ELEMENT],
) (out []string) {
	out = slices.Collect(Strings(slices.Values(collections)))

	sort.Strings(out)

	return out
}

func StringDelimiterSeparated[ELEMENT interfaces.Stringer](
	delimiter string,
	collections ...interfaces.Collection[ELEMENT],
) string {
	if collections == nil {
		return ""
	}

	sorted := SortedStrings(collections...)

	if len(sorted) == 0 {
		return ""
	}

	sb := &strings.Builder{}
	first := true

	for _, e1 := range sorted {
		if !first {
			sb.WriteString(delimiter)
		}

		sb.WriteString(e1)

		first = false
	}

	return sb.String()
}

func StringCommaSeparated[ELEMENT interfaces.Stringer](
	collections ...interfaces.Collection[ELEMENT],
) string {
	return StringDelimiterSeparated(", ", collections...)
}

func StringSpaceSeparated[ELEMENT interfaces.Stringer](
	collections ...interfaces.Collection[ELEMENT],
) string {
	return StringDelimiterSeparated(" ", collections...)
}

func ReverseSortable(sortable sort.Interface) {
	max := sortable.Len() / 2

	for i := range max {
		sortable.Swap(i, sortable.Len()-1-i)
	}
}
