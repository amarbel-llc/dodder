package object_probe_index

import (
	"bytes"
	"fmt"
	"io"

	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

// TODO change to encoder
func (page *page) readRowFrom(
	row *row,
	reader io.Reader,
) (n int64, err error) {
	n, err = row.ReadFromWithHashFormat(reader, page.hashType)

	if err == io.EOF {
		return n, err
	} else if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	if err = markl.MakeErrLength(int64(page.rowWidth), n); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

// TODO change to encoder
func (page *page) writeRowTo(
	row *row,
	writer io.Writer,
) (n int64, err error) {
	n, err = row.WriteTo(writer)

	if err == io.EOF {
		err = nil
	} else if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	if err = markl.MakeErrLength(int64(page.rowWidth), n); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

type row struct {
	Digest markl.Id
	Loc
}

func (row *row) IsEmpty() bool {
	return row.Loc.IsEmpty() && row.Digest.IsNull()
}

func (row *row) String() string {
	return fmt.Sprintf(
		"%s %s %s %x",
		&row.Loc,
		row.Digest.GetPurposeId(),
		row.Digest.GetMarklFormat().GetMarklFormatId(),
		row.Digest.GetBytes(),
	)
}

func (row *row) ReadFromWithHashFormat(
	reader io.Reader,
	hashFormat markl.FormatHash,
) (n int64, err error) {
	var n1 int
	var n2 int64

	n1, err = markl.ReadFrom(reader, &row.Digest, hashFormat)
	n += int64(n1)

	if err == io.EOF {
		return n, err
	} else if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n2, err = row.Loc.ReadFrom(reader)
	n += int64(n2)

	if err == io.EOF {
		return n, err
	} else if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (row *row) WriteTo(writer io.Writer) (n int64, err error) {
	var n1 int
	var n2 int64

	n1, err = writer.Write(row.Digest.GetBytes())
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n2, err = row.Loc.WriteTo(writer)
	n += int64(n2)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

type rowEqualerComplete struct{}

func (rowEqualerComplete) Equals(left, right *row) bool {
	return markl.Equals(&left.Digest, &right.Digest) &&
		left.Loc.Page == right.Loc.Page &&
		left.Loc.Offset == right.Loc.Offset &&
		left.Loc.ContentLength == right.Loc.ContentLength
}

type rowEqualerDigestOnly struct{}

func (rowEqualerDigestOnly) Equals(left, right *row) bool {
	return markl.Equals(&left.Digest, &right.Digest)
}

type rowResetter struct{}

func (rowResetter) Reset(a *row) {
	a.Digest.Reset()
	a.Page = 0
	a.Offset = 0
	a.ContentLength = 0
}

func (rowResetter) ResetWith(a, b *row) {
	a.Digest.ResetWith(b.Digest)
	a.Page = b.Page
	a.Offset = b.Offset
	a.ContentLength = b.ContentLength
}

type rowLessor struct{}

func (rowLessor) Less(left, right *row) bool {
	cmp := bytes.Compare(left.Digest.GetBytes(), right.Digest.GetBytes())

	if cmp != 0 {
		return cmp == -1
	}

	if left.Page != right.Page {
		return left.Page < right.Page
	}

	if left.Offset != right.Offset {
		return left.Offset < right.Offset
	}

	return left.ContentLength < right.ContentLength
}
