package pool

import "code.linenisgreat.com/dodder/go/src/_/interfaces"

type Bespoke[T any] struct {
	FuncGet func() T
	FuncPut func(T)
}

func (ip Bespoke[T]) get() T {
	return ip.FuncGet()
}

func (pool Bespoke[SWIMMER]) GetWithRepool() (SWIMMER, interfaces.FuncRepool) {
	element := pool.get()

	return element, func() {
		pool.put(element)
	}
}

func (ip Bespoke[T]) put(i T) {
	ip.FuncPut(i)
}
