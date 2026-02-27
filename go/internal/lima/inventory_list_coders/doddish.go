package inventory_list_coders

import (
	"bufio"
	"fmt"
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/box_format"
)

type doddish struct {
	box *box_format.BoxTransacted
}

func (coder doddish) EncodeTo(
	object *sku.Transacted,
	bufferedWriter *bufio.Writer,
) (n int64, err error) {
	var n1 int64
	var n2 int

	n1, err = coder.box.EncodeStringTo(object, bufferedWriter)
	n += n1

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n2, err = fmt.Fprintf(bufferedWriter, "\n")
	n += int64(n2)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (coder doddish) DecodeFrom(
	object *sku.Transacted,
	bufferedReader *bufio.Reader,
) (n int64, err error) {
	if n, err = coder.box.ReadStringFormat(object, bufferedReader); err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF)
		return n, err
	}

	return n, err
}
