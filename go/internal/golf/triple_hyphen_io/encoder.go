package triple_hyphen_io

import (
	"bufio"
	"io"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type Encoder[BLOB any] struct {
	Metadata, Blob interfaces.EncoderToBufferedWriter[BLOB]
}

func (coder Encoder[BLOB]) EncodeTo(
	object BLOB,
	writer io.Writer,
) (n int64, err error) {
	bufferedWriter := bufio.NewWriter(writer)
	defer errors.DeferredFlusher(&err, bufferedWriter)

	var n1 int64
	var n2 int

	hasMetadataContent := true

	if mwt, ok := coder.Metadata.(MetadataWriterTo); ok {
		hasMetadataContent = mwt.HasMetadataContent()
	}

	if coder.Metadata != nil && hasMetadataContent {
		n2, err = bufferedWriter.WriteString(Boundary + "\n")
		n += int64(n2)

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		n1, err = coder.Metadata.EncodeTo(object, bufferedWriter)
		n += n1

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		bufferedWriter.WriteString(Boundary + "\n")
		n += n1

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		bufferedWriter.WriteString("\n")
		n += n1

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	if coder.Blob != nil {
		n1, err = coder.Blob.EncodeTo(object, bufferedWriter)
		n += n1

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	return n, err
}
