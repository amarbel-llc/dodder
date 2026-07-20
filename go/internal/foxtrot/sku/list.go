package sku

import (
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

type (
	Seq = interfaces.SeqError[*Transacted]

	InventoryListStore interface {
		WriteInventoryListObject(*Transacted) (err error)
		ReadLast() (max *Transacted, err error)
		AllInventoryListContents(mad_domain_interfaces.MarklId) Seq
		AllInventoryLists() Seq
	}
)
