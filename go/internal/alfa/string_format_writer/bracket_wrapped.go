package string_format_writer

import (
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

func MakeBracketWrapped[T any](
	sfw interfaces.StringEncoderTo[T],
) interfaces.StringEncoderTo[T] {
	return &bracketWrapped[T]{
		stringFormatWriter: sfw,
	}
}

type bracketWrapped[T any] struct {
	stringFormatWriter interfaces.StringEncoderTo[T]
}

func (f bracketWrapped[T]) EncodeStringTo(
	e T,
	w interfaces.WriterAndStringWriter,
) (n int64, err error) {
	var (
		n1 int
		n2 int64
	)

	n1, err = w.WriteString("[")
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n2, err = f.stringFormatWriter.EncodeStringTo(e, w)
	n += int64(n2)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n1, err = w.WriteString("]")
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
