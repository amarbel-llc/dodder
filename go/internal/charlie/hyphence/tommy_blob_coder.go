package hyphence

import (
	"io"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

type TommyBlobDecoder[
	BLOB any,
	BLOB_PTR interfaces.Ptr[BLOB],
] struct {
	Decode func([]byte) (BLOB, error)
}

func (d TommyBlobDecoder[BLOB, BLOB_PTR]) DecodeFrom(
	blob BLOB_PTR,
	reader io.Reader,
) (n int64, err error) {
	var b []byte

	if b, err = io.ReadAll(reader); err != nil {
		err = errors.Wrap(err)
		return
	}

	n = int64(len(b))

	var decoded BLOB

	if decoded, err = d.Decode(b); err != nil {
		err = errors.Wrap(err)
		return
	}

	*blob = decoded

	return
}

type TommyBlobEncoder[
	BLOB any,
	BLOB_PTR interfaces.Ptr[BLOB],
] struct {
	Encode func(BLOB) ([]byte, error)
}

func (e TommyBlobEncoder[BLOB, BLOB_PTR]) EncodeTo(
	blob BLOB_PTR,
	writer io.Writer,
) (n int64, err error) {
	var b []byte

	if b, err = e.Encode(*blob); err != nil {
		err = errors.Wrap(err)
		return
	}

	var written int

	if written, err = writer.Write(b); err != nil {
		err = errors.Wrap(err)
		return
	}

	n = int64(written)

	return
}
