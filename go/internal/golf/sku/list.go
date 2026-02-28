package sku

import (
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

type (
	Seq = interfaces.SeqError[*Transacted]

	InventoryListStore interface {
		WriteInventoryListObject(*Transacted) (err error)
		ReadLast() (max *Transacted, err error)
		AllInventoryListContents(domain_interfaces.MarklId) Seq
		AllInventoryLists() Seq
	}
)
