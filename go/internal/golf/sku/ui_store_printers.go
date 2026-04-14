package sku

import "github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"

type UIStorePrinters struct {
	TransactedNew       interfaces.FuncIter[*Transacted]
	TransactedUpdated   interfaces.FuncIter[*Transacted]
	TransactedUnchanged interfaces.FuncIter[*Transacted]

	CheckedOut interfaces.FuncIter[SkuType] // for when objects are checked out
}
