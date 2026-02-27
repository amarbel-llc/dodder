package quiter

import "code.linenisgreat.com/dodder/go/internal/_/interfaces"

type AsyncSerializer[T any] struct {
	chError <-chan error
	chE     chan<- T
	chDone  <-chan struct{}
}

func MakeAsyncSerializer[T any](
	wf interfaces.FuncIter[T],
) AsyncSerializer[T] {
	chError := make(chan error)
	chE := make(chan T)
	chDone := make(chan struct{})

	go func(chError chan<- error, chE <-chan T, chDone chan<- struct{}) {
		defer func() {
			chDone <- struct{}{}
		}()

		for e := range chE {
			if err := wf(e); err != nil {
				chError <- err
			}
		}
	}(chError, chE, chDone)

	return AsyncSerializer[T]{
		chError: chError,
		chE:     chE,
		chDone:  chDone,
	}
}

func (s AsyncSerializer[T]) Do(e T) (err error) {
	select {
	case err = <-s.chError:
	case s.chE <- e:
	}

	return err
}

func (s AsyncSerializer[T]) Wait() (err error) {
	err = <-s.chError

	close(s.chE)
	<-s.chDone

	return err
}
