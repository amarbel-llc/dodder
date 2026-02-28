package format

import (
	"bufio"
	"fmt"
	"io"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type LineWriter struct {
	lastWasNewline bool
	elements       []interfaces.FuncWriter
}

var MakeLineWriter = NewLineWriter

func NewLineWriter() *LineWriter {
	w := &LineWriter{
		elements: make([]interfaces.FuncWriter, 0),
	}

	return w
}

func (w *LineWriter) WriteTo(out io.Writer) (n int64, err error) {
	w1 := bufio.NewWriter(out)
	defer errors.Deferred(&err, w1.Flush)

	var n1 int64
	var n2 int

	for _, l := range w.elements {
		if n1, err = l(w1); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		n += n1

		if n2, err = io.WriteString(w1, "\n"); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		n += int64(n2)
	}

	return n, err
}

func (w *LineWriter) WriteExactlyOneEmpty() {
	if len(w.elements) == 0 || !w.lastWasNewline {
		w.WriteEmpty()
		return
	}

	return
}

func (w *LineWriter) WriteEmpty() {
	w.lastWasNewline = true

	w.elements = append(
		w.elements,
		func(_ io.Writer) (_ int64, _ error) {
			return
		},
	)
}

func (w *LineWriter) WriteLines(ls ...string) {
	w.lastWasNewline = false

	for _, v := range ls {
		w.elements = append(
			w.elements,
			MakeFormatString("%s", v),
		)
	}
}

func (w *LineWriter) WriteStringers(ss ...fmt.Stringer) {
	w.lastWasNewline = false

	for _, v := range ss {
		w.elements = append(
			w.elements,
			MakeStringer(v),
		)
	}
}

func (w *LineWriter) WriteKeySpaceValue(key, value interface{}) {
	w.lastWasNewline = false

	w.elements = append(
		w.elements,
		MakeFormatString("%s %s", key, value),
	)
}

func (w *LineWriter) WriteFormat(f string, values ...interface{}) {
	w.lastWasNewline = false

	w.elements = append(
		w.elements,
		MakeFormatString(f, values...),
	)
}

func (w *LineWriter) WriteFormats(f string, values ...interface{}) {
	w.lastWasNewline = false

	for _, v := range values {
		w.elements = append(
			w.elements,
			MakeFormatString(f, v),
		)
	}
}
