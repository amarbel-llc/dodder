package collections_ptr

import (
	"encoding/gob"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/quiter"
)

func RegisterGobValue[
	VALUE interfaces.Value,
	VALUE_PTR interfaces.ValuePtr[VALUE],
](
	keyer interfaces.StringKeyerPtr[VALUE, VALUE_PTR],
) {
	if keyer == nil {
		keyer = quiter.StringerKeyerPtr[VALUE, VALUE_PTR]{}.RegisterGob()
	}

	gob.Register(keyer)

	RegisterGob[VALUE, VALUE_PTR]()
}

func RegisterGob[
	VALUE interfaces.Value,
	VALUE_PTR interfaces.ValuePtr[VALUE],
]() {
	gob.Register(Set[VALUE, VALUE_PTR]{})
	gob.Register(MutableSet[VALUE, VALUE_PTR]{})
}
