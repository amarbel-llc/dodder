package format

import (
	"fmt"
	"io"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func MakeWriter[ELEMENT any](
	wff interfaces.FuncWriterFormat[ELEMENT],
	e ELEMENT,
) interfaces.FuncWriter {
	return func(w io.Writer) (int64, error) {
		return wff(w, e)
	}
}

func MakeWriterOr[A interfaces.Stringer, B interfaces.Stringer](
	wffA interfaces.FuncWriterFormat[A],
	eA A,
	wffB interfaces.FuncWriterFormat[B],
	eB B,
) interfaces.FuncWriter {
	return func(w io.Writer) (int64, error) {
		if eA.String() == "" {
			return wffB(w, eB)
		} else {
			return wffA(w, eA)
		}
	}
}

func MakeWriterPtr[ELEMENT any](
	wff interfaces.FuncWriterFormat[*ELEMENT],
	e *ELEMENT,
) interfaces.FuncWriter {
	return func(w io.Writer) (int64, error) {
		return wff(w, e)
	}
}

func MakeFormatString(
	f string,
	vs ...interface{},
) interfaces.FuncWriter {
	return func(w io.Writer) (n int64, err error) {
		var n1 int

		if n1, err = io.WriteString(w, fmt.Sprintf(f, vs...)); err != nil {
			n = int64(n1)
			err = errors.Wrap(err)
			return n, err
		}

		n = int64(n1)

		return n, err
	}
}

func MakeStringer(
	v fmt.Stringer,
) interfaces.FuncWriter {
	return func(w io.Writer) (n int64, err error) {
		var n1 int

		if n1, err = io.WriteString(w, v.String()); err != nil {
			n = int64(n1)
			err = errors.Wrap(err)
			return n, err
		}

		n = int64(n1)

		return n, err
	}
}

func MakeFormatStringer[ELEMENT interfaces.Stringer](
	sf interfaces.FuncString[interfaces.Set[ELEMENT]],
) interfaces.FuncWriterFormat[interfaces.Set[ELEMENT]] {
	return func(w io.Writer, e interfaces.Set[ELEMENT]) (n int64, err error) {
		var n1 int

		if n1, err = io.WriteString(w, sf(e)); err != nil {
			n = int64(n1)
			err = errors.Wrap(err)
			return n, err
		}

		n = int64(n1)

		return n, err
	}
}
