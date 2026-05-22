package collections_coding

import "github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"

type EncoderLike[T any] interface {
	Encode(*T) (int64, error)
}

func EncoderToWriter[T any](el EncoderLike[T]) interfaces.FuncIter[*T] {
	return func(e *T) (err error) {
		_, err = el.Encode(e)
		return err
	}
}
