package alfred

import (
	"sync"

	"code.linenisgreat.com/dodder/go/lib/delta/catgut"
)

type ItemPool struct {
	inner *sync.Pool
}

func MakeItemPool() ItemPool {
	return ItemPool{
		inner: &sync.Pool{
			New: func() interface{} {
				return &Item{
					Match: &catgut.String{},
					Mods:  make(map[string]Mod),
				}
			},
		},
	}
}

func (ip ItemPool) Get() *Item {
	return ip.inner.Get().(*Item)
}

func (ip ItemPool) Put(i *Item) {
	if i == nil {
		panic("item was nil")
	}

	i.Reset()
	ip.inner.Put(i)
}
