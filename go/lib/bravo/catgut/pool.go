package catgut

import (
	"sync"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/pool"
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
