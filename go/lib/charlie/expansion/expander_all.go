package expansion

import (
	"regexp"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

type expanderAll struct {
	delimiter *regexp.Regexp
}

var _ Expander = expanderAll{}

func MakeExpanderAll(
	delimiter string,
) expanderAll {
	return expanderAll{
		delimiter: regexp.MustCompile(delimiter),
	}
}

func (expander expanderAll) Expand(
	value string,
) interfaces.Seq[string] {
	return func(yield func(string) bool) {
		if !yield(value) {
			return
		}

		if value == "" {
			return
		}

		delim := expander.delimiter.FindAllIndex([]byte(value), -1)

		if delim == nil {
			return
		}

		end := len(value)
		prevRight := 0

		for index, loc := range delim {
			left := loc[0]
			right := loc[1]

			t1 := value[0:left]
			t2 := value[right:end]

			if !yield(t1) {
				return
			}

			if !yield(t2) {
				return
			}

			if 0 < index && index < len(delim) {
				t1 := value[prevRight:left]

				if !yield(t1) {
					return
				}
			}

			prevRight = right
		}
	}
}
