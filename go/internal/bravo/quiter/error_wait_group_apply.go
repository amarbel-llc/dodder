package quiter

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
)

func ErrorWaitGroupApply[T any](
	wg errors.WaitGroup,
	s interfaces.Collection[T],
	f interfaces.FuncIter[T],
) bool {
	for e := range s.All() {
		if !wg.Do(
			func() error {
				return f(e)
			},
		) {
			return true
		}
	}

	return false
}
