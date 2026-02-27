package inventory_list_coders

import (
	"bufio"
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
)

type coder struct {
	listCoder      sku.ListCoder
	beforeEncoding func(*sku.Transacted) error
	afterDecoding  func(*sku.Transacted) error
}

func (coder coder) EncodeTo(
	object *sku.Transacted,
	bufferedWriter *bufio.Writer,
) (n int64, err error) {
	if coder.beforeEncoding != nil {
		if err = coder.beforeEncoding(object); err != nil {
			err = errors.Wrapf(err, "Object: %s", sku.String(object))
			return n, err
		}
	}

	if n, err = coder.listCoder.EncodeTo(object, bufferedWriter); err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF)
		return n, err
	}

	return n, err
}

func (coder coder) DecodeFrom(
	object *sku.Transacted,
	bufferedReader *bufio.Reader,
) (n int64, err error) {
	var eof bool

	if n, err = coder.listCoder.DecodeFrom(object, bufferedReader); err != nil {
		if err == io.EOF {
			eof = true

			if n == 0 {
				return n, err
			}
		} else {
			err = errors.WrapExceptSentinel(err, io.EOF)
			return n, err
		}
	}

	if coder.afterDecoding != nil {
		if err = coder.afterDecoding(object); err != nil {
			err = ErrAfterDecoding{err: err, object: object}
			return n, err
		}
	}

	if eof {
		err = io.EOF
	}

	return n, err
}
