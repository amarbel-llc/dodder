package ids

import (
	"encoding/gob"
	"sync"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/collections_ptr"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
)

// TODO remove this once gob is removed entirely

var (
	registerOnce   sync.Once
	registryLock   *sync.Mutex
	registryGenres map[genres.Genre]Id
)

func once() {
	registryLock = &sync.Mutex{}
	registryGenres = make(map[genres.Genre]Id)
}

func register[T Id, TPtr interface {
	interfaces.StringSetterPtr[T]
	Id
}](id T,
) {
	gob.Register(&id)
	gob.Register(collections_ptr.MakeMutableValueSet[T, TPtr](nil))
	gob.Register(collections_ptr.MakeValueSet[T, TPtr](nil))
	registerOnce.Do(once)

	registryLock.Lock()
	defer registryLock.Unlock()

	ok := false
	var id1 Id
	g := genres.Must(id.GetGenre())

	if id1, ok = registryGenres[g]; ok {
		panic(
			errors.ErrorWithStackf(
				"genre %s has two registrations: %s (old), %s (new)",
				g,
				id1,
				id,
			),
		)
	}

	registryGenres[g] = id
}
