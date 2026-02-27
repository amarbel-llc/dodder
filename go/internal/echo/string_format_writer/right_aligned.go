package string_format_writer

import (
	"strings"
	"unicode/utf8"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func MakeRightAligned() interfaces.StringEncoderTo[string] {
	return &rightAligned{}
}

type rightAligned struct{}

func (f rightAligned) EncodeStringTo(
	v string,
	w interfaces.WriterAndStringWriter,
) (n int64, err error) {
	diff := LenStringMax + 1 - utf8.RuneCountInString(v)

	if diff > 0 {
		v = strings.Repeat(" ", diff-1) + v
	}

	var n1 int

	n1, err = w.WriteString(v)
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n1, err = w.WriteString(" ")
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

type rightAligned2 struct {
	interfaces.WriterAndStringWriter
}

func (w rightAligned2) Write(b []byte) (n int, err error) {
	var n1 int

	diff := LenStringMax - utf8.RuneCount(b)

	if diff > 0 {
		space := strings.Repeat(" ", diff)
		n1, err = w.WriterAndStringWriter.WriteString(space)
		n += n1

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	n1, err = w.WriterAndStringWriter.Write(b)
	n += n1

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n1, err = w.WriterAndStringWriter.WriteString(" ")
	n += n1

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (w rightAligned2) WriteString(v string) (n int, err error) {
	var n1 int

	diff := LenStringMax + 1 - utf8.RuneCountInString(v)

	if diff > 0 {
		space := strings.Repeat(" ", diff-1)
		n1, err = w.WriterAndStringWriter.WriteString(space)
		n += n1

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	n1, err = w.WriterAndStringWriter.WriteString(v)
	n += n1

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n1, err = w.WriterAndStringWriter.WriteString(" ")
	n += n1

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
