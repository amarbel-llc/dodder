package catgut

import (
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

type stringFormatWriter struct{}

var StringFormatWriterString stringFormatWriter

func (stringFormatWriter) EncodeStringTo(
	e *String,
	sw interfaces.WriterAndStringWriter,
) (n int64, err error) {
	n, err = e.WriteTo(sw)
	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
