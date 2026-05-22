package sku

import (
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
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
