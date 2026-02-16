package pool

import "code.linenisgreat.com/dodder/go/src/_/interfaces"

type fakePool[SWIMMER any, SWIMMER_PTR interfaces.Ptr[SWIMMER]] struct{}

var _ interfaces.PoolPtr[string, *string] = fakePool[string, *string]{}

func MakeFakePool[T any, TPtr interfaces.Ptr[T]]() *fakePool[T, TPtr] {
	return &fakePool[T, TPtr]{}
}

func (pool fakePool[T, TPtr]) get() TPtr {
	var t T
	return &t
}

func (pool fakePool[SWIMMER, SWIMMER_PTR]) GetWithRepool() (SWIMMER_PTR, interfaces.FuncRepool) {
	element := pool.get()
	return element, wrapRepoolDebug(func() {})
}

func (pool fakePool[T, TPtr]) put(i TPtr) {}
