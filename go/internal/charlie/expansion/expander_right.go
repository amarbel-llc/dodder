package expansion

import (
	"regexp"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
)

type expanderRight struct {
	delimiter *regexp.Regexp
}

var _ Expander = expanderRight{}

func MakeExpanderRight(
	delimiter string,
) expanderRight {
	return expanderRight{
		delimiter: regexp.MustCompile(delimiter),
	}
}

func (expander expanderRight) Expand(
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

		for _, loc := range delim {
			locStart := loc[0]
			t1 := value[0:locStart]

			if !yield(t1) {
				return
			}
		}
	}
}
