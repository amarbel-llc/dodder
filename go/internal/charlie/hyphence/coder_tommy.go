package hyphence

import (
	"bufio"
	"io"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type CoderTommy[
	BLOB any,
	BLOB_PTR interfaces.Ptr[BLOB],
] struct {
	Decode func([]byte) (BLOB, error)
	Encode func(BLOB) ([]byte, error)
}

func (coder CoderTommy[BLOB, BLOB_PTR]) DecodeFrom(
	blob BLOB_PTR,
	bufferedReader *bufio.Reader,
) (n int64, err error) {
	var input []byte

	if input, err = io.ReadAll(bufferedReader); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n = int64(len(input))

	var decoded BLOB

	if decoded, err = coder.Decode(input); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	*blob = decoded

	return n, err
}

func (coder CoderTommy[BLOB, BLOB_PTR]) EncodeTo(
	blob BLOB_PTR,
	bufferedWriter *bufio.Writer,
) (n int64, err error) {
	var output []byte

	if output, err = coder.Encode(*blob); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	var nInt int

	if nInt, err = bufferedWriter.Write(output); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n = int64(nInt)

	return n, err
}
