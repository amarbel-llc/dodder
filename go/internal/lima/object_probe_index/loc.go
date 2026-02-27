package object_probe_index

import (
	"fmt"
	"io"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
)

type Loc struct {
	Page uint8
	ohio.Cursor
}

func (loc Loc) IsEmpty() bool {
	return loc.Page == 0 && loc.Offset == 0 && loc.ContentLength == 0
}

func (loc Loc) String() string {
	return fmt.Sprintf("%02d@%s", loc.Page, loc.Cursor)
}

func (loc *Loc) ReadFrom(reader io.Reader) (n int64, err error) {
	var n1 int
	loc.Page, n1, err = ohio.ReadFixedUint8(reader)
	n += int64(n1)

	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}

		err = errors.Wrap(err)
		return n, err
	}

	var n2 int64
	n2, err = loc.Cursor.ReadFrom(reader)
	n += n2

	if err == io.EOF {
		return n, err
	} else if err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF)
		return n, err
	}

	return n, err
}

func (loc *Loc) WriteTo(writer io.Writer) (n int64, err error) {
	var n1 int
	n1, err = ohio.WriteUint8(writer, loc.Page)
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	var n2 int64
	n2, err = loc.Cursor.WriteTo(writer)
	n += n2

	if err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF)
		return n, err
	}

	return n, err
}
