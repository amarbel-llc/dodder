package markl

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
)

var idPool interfaces.PoolPtr[Id, *Id] = pool.MakeWithResetable[Id]()

func GetId() (domain_interfaces.MarklIdMutable, interfaces.FuncRepool) {
	return idPool.GetWithRepool()
}
