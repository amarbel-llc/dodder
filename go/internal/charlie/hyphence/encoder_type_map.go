package hyphence

import (
	"bufio"

	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type EncoderTypeMapWithoutType[BLOB any] map[string]interfaces.EncoderToBufferedWriter[BLOB]

func (coderTypeMap EncoderTypeMapWithoutType[BLOB]) EncodeTo(
	typedBlob *TypedBlob[BLOB],
	bufferedWriter *bufio.Writer,
) (n int64, err error) {
	tipe := typedBlob.Type
	coder, ok := coderTypeMap[tipe.String()]

	if !ok {
		err = errors.ErrorWithStackf("no coders available for type: %q", tipe)
		return n, err
	}

	if n, err = coder.EncodeTo(typedBlob.Blob, bufferedWriter); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
