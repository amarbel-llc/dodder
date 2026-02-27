package triple_hyphen_io

import (
	"bufio"
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/ohio"
)

type CoderToTypedBlob[BLOB any] struct {
	RequireMetadata bool
	// TODO replace with a metadata-like object
	Metadata interfaces.CoderBufferedReadWriter[*TypedBlob[BLOB]]
	Blob     CoderTypeMapWithoutType[BLOB]
}

func (coder CoderToTypedBlob[O]) DecodeFrom(
	typedBlob *TypedBlob[O],
	reader io.Reader,
) (n int64, err error) {
	var n1 int64
	n1, err = coder.readMetadataFrom(typedBlob, &reader)
	n += n1

	if err != nil {
		err = errors.Wrapf(err, "metadata read failed")
		return n, err
	}

	n1, err = coder.Blob.DecodeFrom(typedBlob, bufio.NewReader(reader))
	n += n1

	if err != nil {
		err = errors.Wrapf(err, "blob read failed for type: %q", typedBlob.Type)
		return n, err
	}

	return n, err
}

func (coder CoderToTypedBlob[O]) readMetadataFrom(
	typedBlob *TypedBlob[O],
	reader *io.Reader,
) (n int64, err error) {
	var state readerState
	bufferedReader := bufio.NewReader(*reader)

	if coder.RequireMetadata && coder.Metadata == nil {
		err = errors.ErrorWithStackf("metadata reader is nil")
		return n, err
	}

	if coder.Blob == nil {
		err = errors.ErrorWithStackf("blob reader is nil")
		return n, err
	}

	var metadataPipe ohio.PipedReader

	isEOF := false

LINE_READ_LOOP:
	for !isEOF {
		var rawLine, line string

		rawLine, err = bufferedReader.ReadString('\n')
		n += int64(len(rawLine))

		if err != nil && err != io.EOF {
			err = errors.Wrap(err)
			return n, err
		}

		if errors.IsEOF(err) {
			err = nil
			isEOF = true
		}

		line = strings.TrimSuffix(rawLine, "\n")

		switch state {
		case readerStateEmpty:
			switch {
			case coder.RequireMetadata && line != Boundary:
				err = errors.ErrorWithStackf("expected %q but got %q", Boundary, line)
				return n, err

			case line != Boundary:
				*reader = io.MultiReader(
					strings.NewReader(rawLine),
					bufferedReader,
				)

				break LINE_READ_LOOP
			}

			state += 1

			metadataPipe = ohio.MakePipedDecoder(typedBlob, coder.Metadata)

		case readerStateFirstBoundary:
			if line == Boundary {
				if _, err = metadataPipe.Close(); err != nil {
					err = errors.Wrapf(err, "metadata read failed")
					return n, err
				}

				state += 1
				break
			}

			if _, err = metadataPipe.Write([]byte(rawLine)); err != nil {
				err = errors.Wrap(err)
				return n, err
			}

		case readerStateSecondBoundary:
			*reader = bufferedReader
			break LINE_READ_LOOP

		default:
			err = errors.ErrorWithStackf("impossible state %d", state)
			return n, err
		}
	}

	return n, err
}

func (coder CoderToTypedBlob[O]) EncodeTo(
	typedBlob *TypedBlob[O],
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

		n1, err = coder.Metadata.EncodeTo(typedBlob, bufferedWriter)
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
		n1, err = coder.Blob.EncodeTo(typedBlob, bufferedWriter)
		n += n1

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	return n, err
}
