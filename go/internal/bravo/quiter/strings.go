package quiter

import (
	"slices"
	"sort"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/cmp"
)

// TODO move to collections_slice
func SortedValuesBy[ELEMENT any](
	set interfaces.Collection[ELEMENT],
	cmp cmp.Func[ELEMENT],
) (out []ELEMENT) {
	out = CollectSlice(set)

	sort.Slice(
		out,
		func(left, right int) bool {
			return cmp(out[left], out[right]).IsLess()
		},
	)

	return out
}

// TODO move to collections_slice
func SortedValues[ELEMENT interfaces.Value](
	seq interfaces.Seq[ELEMENT],
) (out []ELEMENT) {
	out = slices.Collect(seq)

	sort.Slice(
		out,
		func(i, j int) bool { return out[i].String() < out[j].String() },
	)

	return out
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

// TODO move to collections_slice
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

func ReverseSortable(sortable sort.Interface) {
	max := sortable.Len() / 2

	for i := range max {
		sortable.Swap(i, sortable.Len()-1-i)
	}
}
