package store_abbr

import (
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

type indexNoAbbr[
	ID interfaces.Stringer,
	ID_PTR interfaces.SetterPtr[ID],
] struct {
	sku.IdAbbrIndexGeneric[ID, ID_PTR]
}

func (ih indexNoAbbr[ID, ID_PTR]) Abbreviate(h ID) (v string, err error) {
	v = h.String()
	return v, err
}
