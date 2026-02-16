package pool

import (
	"sync"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
)

type Slice[SWIMMER any, SWIMMER_SLICE ~[]SWIMMER] struct {
	inner *sync.Pool
}

func MakeSlice[SWIMMER any, SWIMMER_SLICE ~[]SWIMMER]() Slice[SWIMMER, SWIMMER_SLICE] {
	return Slice[SWIMMER, SWIMMER_SLICE]{
		inner: &sync.Pool{
			New: func() any {
				swimmer := make(SWIMMER_SLICE, 0)
				return swimmer
			},
		},
	}
}

func (pool Slice[_, SWIMMER_SLICE]) get() SWIMMER_SLICE {
	return pool.inner.Get().(SWIMMER_SLICE)
}

func (pool Slice[_, SWIMMER_SLICE]) GetWithRepool() (SWIMMER_SLICE, interfaces.FuncRepool) {
	element := pool.get()
	return element, func() {
		pool.put(element)
	}
}

func (pool Slice[_, SWIMMER_SLICE]) put(swimmer SWIMMER_SLICE) {
	swimmer = swimmer[:0]
	pool.inner.Put(swimmer)
}
