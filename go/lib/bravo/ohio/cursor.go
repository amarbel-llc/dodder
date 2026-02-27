package ohio

import (
	"fmt"
	"io"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

type Cursor struct {
	Offset, ContentLength int64
}

func (cursor *Cursor) Size() int {
	return 8 * 2
}

func (cursor Cursor) IsEmpty() bool {
	return cursor.Offset == 0 && cursor.ContentLength == 0
}

func (cursor Cursor) String() string {
	return fmt.Sprintf("%03d+%03d", cursor.Offset, cursor.ContentLength)
}

func (cursor *Cursor) Reset() {
	cursor.Offset = 0
	cursor.ContentLength = 0
}

func (cursor *Cursor) ReadFrom(r io.Reader) (n int64, err error) {
	var n1 int
	n1, cursor.Offset, err = ReadFixedInt64(r)
	n += int64(n1)

	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}

		err = errors.Wrap(err)
		return n, err
	}

	n1, cursor.ContentLength, err = ReadFixedInt64(r)
	n += int64(n1)

	if err == io.EOF {
		return n, err
	} else if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (cursor *Cursor) WriteTo(w io.Writer) (n int64, err error) {
	var n1 int
	n1, err = WriteFixedInt64(w, cursor.Offset)
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n1, err = WriteFixedInt64(w, cursor.ContentLength)
	n += int64(n1)

	if err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF)
		return n, err
	}

	return n, err
}
