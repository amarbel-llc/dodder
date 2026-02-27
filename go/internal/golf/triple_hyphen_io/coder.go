package triple_hyphen_io

import (
	"bufio"
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
)

type Coder[BLOB any] struct {
	RequireMetadata bool
	Metadata, Blob  interfaces.CoderBufferedReadWriter[BLOB]
}

func (coder *Coder[O]) DecodeFrom(object O, r io.Reader) (n int64, err error) {
	var n1 int64
	n1, err = coder.readMetadataFrom(object, &r)
	n += n1

	if err != nil {
		err = errors.Wrapf(err, "metadata read failed")
		return n, err
	}

	n1, err = coder.Blob.DecodeFrom(object, bufio.NewReader(r))
	n += n1

	if err != nil {
		err = errors.Wrapf(err, "blob read failed")
		return n, err
	}

	return n, err
}

func (coder *Coder[O]) readMetadataFrom(
	object O,
	bufferedReader *io.Reader,
) (n int64, err error) {
	var state readerState
	br := bufio.NewReader(*bufferedReader)

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

		rawLine, err = br.ReadString('\n')
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
				*bufferedReader = io.MultiReader(
					strings.NewReader(rawLine),
					br,
				)

				break LINE_READ_LOOP
			}

			state += 1

			metadataPipe = ohio.MakePipedDecoder(object, coder.Metadata)

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
			*bufferedReader = br
			break LINE_READ_LOOP

		default:
			err = errors.ErrorWithStackf("impossible state %d", state)
			return n, err
		}
	}

	return n, err
}

func (coder Coder[O]) EncodeTo(
	object O,
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
