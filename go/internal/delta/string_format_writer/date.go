package string_format_writer

import (
	"io"
	"time"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

type Clock interface {
	GetTime() time.Time
}

type Date[T any] struct {
	Clock
	Format string
	interfaces.StringEncoderTo[T]
}

func MakeDefaultDatePrefixFormatWriter[T any](
	clock Clock,
	f interfaces.StringEncoderTo[T],
) interfaces.StringEncoderTo[T] {
	return &Date[T]{
		Clock:           clock,
		Format:          StringFormatDateTime,
		StringEncoderTo: f,
	}
}

func (f *Date[T]) EncodeStringTo(
	e T,
	w interfaces.WriterAndStringWriter,
) (n int64, err error) {
	d := f.GetTime().Format(f.Format)

	var n1 int

	n1, err = io.WriteString(w, d)
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n1, err = io.WriteString(w, " ")
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	var n2 int64

	n2, err = f.StringEncoderTo.EncodeStringTo(e, w)
	n += n2

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
