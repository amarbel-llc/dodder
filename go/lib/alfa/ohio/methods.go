package ohio

import (
	"io"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/pool"
)

func WriteSeq[ELEMENT any](
	writer io.Writer,
	element ELEMENT,
	seq ...interfaces.FuncWriterElementInterface[ELEMENT],
) (n int64, err error) {
	bufferedWriter, repool := pool.GetBufferedWriter(writer)
	defer repool()

	defer errors.DeferredFlusher(&err, bufferedWriter)

	var n1 int64

	for _, funcWrite := range seq {
		n1, err = funcWrite(bufferedWriter, element)

		n += n1

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	return n, err
}

// TODO-P4 check performance of this
func WriteLine(writer io.Writer, value string) (n int64, err error) {
	var n1 int

	if value != "" {
		n1, err = io.WriteString(writer, value)

		n += int64(n1)

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	n1, err = io.WriteString(writer, "\n")

	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
