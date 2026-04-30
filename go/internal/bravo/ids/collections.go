package ids

import (
	"code.linenisgreat.com/dodder/go/lib/bravo/collections_slice"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

type (
	Slice[ELEMENT any] = collections_slice.Slice[ELEMENT]

	Set[ELEMENT any] interface {
		Len() int
		All() interfaces.Seq[ELEMENT]
		ContainsKey(string) bool
		Get(string) (ELEMENT, bool)
		Key(ELEMENT) string
	}

	SetMutable[ELEMENT any] = interface {
		Set[ELEMENT]

		interfaces.Adder[ELEMENT]
		DelKey(string) error
		interfaces.Resetable
	}
)
