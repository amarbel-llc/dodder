package pool

import "code.linenisgreat.com/dodder/go/lib/_/interfaces"

type Bespoke[T any] struct {
	FuncGet func() T
	FuncPut func(T)
}

func (ip Bespoke[T]) get() T {
	return ip.FuncGet()
}

func (pool Bespoke[SWIMMER]) GetWithRepool() (SWIMMER, interfaces.FuncRepool) {
	element := pool.get()

	return element, wrapRepoolDebug(func() {
		pool.put(element)
	})
}

func (ip Bespoke[T]) put(i T) {
	ip.FuncPut(i)
}
