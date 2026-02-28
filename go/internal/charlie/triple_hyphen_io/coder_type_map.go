package triple_hyphen_io

import (
	"bufio"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type TypedBlob[BLOB any] struct {
	// TODO determine why this needs to be a pointer
	Type ids.TypeStruct
	Blob BLOB
}

type TypedBlobEmpty = TypedBlob[struct{}]

type CoderTypeMap[BLOB any] map[string]interfaces.CoderBufferedReadWriter[*TypedBlob[BLOB]]

func (coderTypeMap CoderTypeMap[BLOB]) DecodeFrom(
	typedBlob *TypedBlob[BLOB],
	bufferedReader *bufio.Reader,
) (n int64, err error) {
	tipe := typedBlob.Type
	coder, ok := coderTypeMap[tipe.String()]

	if !ok {
		err = errors.ErrorWithStackf("no coders available for type: %q", tipe)
		return n, err
	}

	if n, err = coder.DecodeFrom(typedBlob, bufferedReader); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (coderTypeMap CoderTypeMap[BLOB]) EncodeTo(
	typedBlob *TypedBlob[BLOB],
	bufferedWriter *bufio.Writer,
) (n int64, err error) {
	tipe := typedBlob.Type
	coder, ok := coderTypeMap[tipe.String()]

	if !ok {
		err = errors.ErrorWithStackf("no coders available for type: %q", tipe)
		return n, err
	}

	if n, err = coder.EncodeTo(typedBlob, bufferedWriter); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

type DecoderTypeMapWithoutType[BLOB any] map[string]interfaces.DecoderFromBufferedReader[BLOB]

func (decoderTypeMap DecoderTypeMapWithoutType[BLOB]) DecodeFrom(
	typedBlob *TypedBlob[BLOB],
	bufferedReader *bufio.Reader,
) (n int64, err error) {
	tipe := typedBlob.Type
	decoder, ok := decoderTypeMap[tipe.String()]

	if !ok {
		err = errors.ErrorWithStackf("no coders available for type: %q", tipe)
		return n, err
	}

	if n, err = decoder.DecodeFrom(
		typedBlob.Blob,
		bufferedReader,
	); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

type CoderTypeMapWithoutType[BLOB any] map[string]interfaces.CoderBufferedReadWriter[*BLOB]

func (coderTypeMap CoderTypeMapWithoutType[BLOB]) DecodeFrom(
	typedBlob *TypedBlob[BLOB],
	bufferedReader *bufio.Reader,
) (n int64, err error) {
	tipe := typedBlob.Type
	coder, ok := coderTypeMap[tipe.String()]

	if !ok {
		err = errors.ErrorWithStackf("no coders available for type: %q", tipe)
		return n, err
	}

	if n, err = coder.DecodeFrom(&typedBlob.Blob, bufferedReader); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (coderTypeMap CoderTypeMapWithoutType[BLOB]) EncodeTo(
	typedBlob *TypedBlob[BLOB],
	bufferedWriter *bufio.Writer,
) (n int64, err error) {
	tipe := typedBlob.Type
	coder, ok := coderTypeMap[tipe.String()]

	if !ok {
		err = errors.ErrorWithStackf("no coders available for type: %q", tipe)
		return n, err
	}

	if n, err = coder.EncodeTo(&typedBlob.Blob, bufferedWriter); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
