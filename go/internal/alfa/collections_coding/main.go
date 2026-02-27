package collections_coding

import "code.linenisgreat.com/dodder/go/internal/_/interfaces"

type EncoderLike[T any] interface {
	Encode(*T) (int64, error)
}

func EncoderToWriter[T any](el EncoderLike[T]) interfaces.FuncIter[*T] {
	return func(e *T) (err error) {
		_, err = el.Encode(e)
		return err
	}
}
