package string_format_writer

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

type quoted_streeng[T ~string] struct{}

func MakeQuotedString[T ~string]() interfaces.StringEncoderTo[T] {
	return &quoted_streeng[T]{}
}

func (f *quoted_streeng[T]) EncodeStringTo(
	e T,
	sw interfaces.WriterAndStringWriter,
) (n int64, err error) {
	var n1 int

	n1, err = fmt.Fprintf(sw, "%q", string(e))
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
