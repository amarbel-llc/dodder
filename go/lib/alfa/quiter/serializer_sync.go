package quiter

import (
	"sync"

	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

func MakeSyncSerializer[ELEMENT any](
	funk interfaces.FuncIter[ELEMENT],
) interfaces.FuncIter[ELEMENT] {
	lock := &sync.Mutex{}

	return func(element ELEMENT) (err error) {
		lock.Lock()
		defer lock.Unlock()

		if err = funk(element); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}
}
