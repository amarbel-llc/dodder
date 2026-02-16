package pool

import (
	"sync"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
)

type value[SWIMMER any] struct {
	inner *sync.Pool
	reset func(SWIMMER)
}

var _ interfaces.Pool[string] = value[string]{}

func MakeValue[SWIMMER any](
	New func() SWIMMER,
	Reset func(SWIMMER),
) *value[SWIMMER] {
	return &value[SWIMMER]{
		reset: Reset,
		inner: &sync.Pool{
			New: func() (swimmer any) {
				if New == nil {
					var element SWIMMER
					swimmer = element
				} else {
					swimmer = New()
				}

				return swimmer
			},
		},
	}
}

func (pool value[SWIMMER]) get() SWIMMER {
	return pool.inner.Get().(SWIMMER)
}

func (pool value[SWIMMER]) GetWithRepool() (SWIMMER, interfaces.FuncRepool) {
	element := pool.get()

	return element, wrapRepoolDebug(func() {
		pool.put(element)
	})
}

func (pool value[SWIMMER]) put(swimmer SWIMMER) {
	if pool.reset != nil {
		pool.reset(swimmer)
	}

	pool.inner.Put(swimmer)
}
