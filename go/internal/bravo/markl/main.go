package markl

import (
	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
)

var idPool interfaces.PoolPtr[Id, *Id] = pool.MakeWithResetable[Id]()

func GetId() (domain_interfaces.MarklIdMutable, interfaces.FuncRepool) {
	return idPool.GetWithRepool()
}
