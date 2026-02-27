package collections_value

import (
	"encoding/gob"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/quiter"
)

func RegisterGobValue[ELEMENT interfaces.Stringer](
	keyer interfaces.StringKeyer[ELEMENT],
) {
	if keyer == nil {
		keyer = quiter.StringerKeyer[ELEMENT]{}.RegisterGob()
	}

	gob.Register(keyer)

	RegisterGob[ELEMENT]()
}

func RegisterGob[ELEMENT interfaces.Stringer]() {
	gob.Register(Set[ELEMENT]{})
	gob.Register(MutableSet[ELEMENT]{})
}
