package catgut

import (
	"sync"

	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
)

var (
	p     interfaces.PoolPtr[String, *String]
	ponce sync.Once
)

func init() {
}

func GetPool() interfaces.PoolPtr[String, *String] {
	ponce.Do(
		func() {
			p = pool.Make[String, *String](
				nil,
				func(v *String) {
					v.Reset()
				},
			)
		},
	)

	return p
}
