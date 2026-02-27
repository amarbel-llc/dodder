package string_format_writer

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
)

type streeng[T ~string] struct{}

func MakeString[T ~string]() interfaces.StringEncoderTo[T] {
	return &streeng[T]{}
}

func (f *streeng[T]) EncodeStringTo(
	e T,
	sw interfaces.WriterAndStringWriter,
) (n int64, err error) {
	var n1 int

	n1, err = sw.WriteString(string(e))
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
