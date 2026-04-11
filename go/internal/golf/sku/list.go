package sku

import (
	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/0/interfaces"
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
