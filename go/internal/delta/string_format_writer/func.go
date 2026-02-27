package string_format_writer

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
)

func MakeFunc[T any](
	f interfaces.FuncStringWriterFormat[T],
) interfaces.StringEncoderTo[T] {
	return funk[T](f)
}

type funk[T any] interfaces.FuncStringWriterFormat[T]

func (f funk[T]) EncodeStringTo(
	e T,
	w interfaces.WriterAndStringWriter,
) (int64, error) {
	return interfaces.FuncStringWriterFormat[T](f)(w, e)
}
