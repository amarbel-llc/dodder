package triple_hyphen_io

import (
	"bufio"
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/ohio"
)

type Reader struct {
	RequireMetadata bool // TODO-P4 add delimiter
	Metadata, Blob  io.ReaderFrom
}

func (reader *Reader) ReadFrom(ioReader io.Reader) (n int64, err error) {
	var n1 int64
	n1, err = reader.readMetadataFrom(&ioReader)
	n += n1

	if err != nil {
		err = errors.Wrapf(err, "metadata read failed")
		return n, err
	}

	n1, err = reader.Blob.ReadFrom(bufio.NewReader(ioReader))
	n += n1

	if err != nil {
		err = errors.Wrapf(err, "blob read failed")
		return n, err
	}

	return n, err
}

func (reader *Reader) readMetadataFrom(ioReader *io.Reader) (n int64, err error) {
	var state readerState
	br := bufio.NewReader(*ioReader)

	if reader.RequireMetadata && reader.Metadata == nil {
		err = errors.ErrorWithStackf("metadata reader is nil")
		return n, err
	}

	if reader.Blob == nil {
		err = errors.ErrorWithStackf("blob reader is nil")
		return n, err
	}

	var object_metadata ohio.PipedReader

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
			case reader.RequireMetadata && line != Boundary:
				err = errors.ErrorWithStackf("expected %q but got %q", Boundary, line)
				return n, err

			case line != Boundary:
				*ioReader = io.MultiReader(
					strings.NewReader(rawLine),
					br,
				)

				break LINE_READ_LOOP
			}

			state += 1

			object_metadata = ohio.MakePipedReaderFrom(reader.Metadata)

		case readerStateFirstBoundary:
			if line == Boundary {
				if _, err = object_metadata.Close(); err != nil {
					err = errors.Wrapf(err, "metadata read failed")
					return n, err
				}

				state += 1
				break
			}

			if _, err = object_metadata.Write([]byte(rawLine)); err != nil {
				err = errors.Wrap(err)
				return n, err
			}

		case readerStateSecondBoundary:
			*ioReader = br
			break LINE_READ_LOOP

		default:
			err = errors.ErrorWithStackf("impossible state %d", state)
			return n, err
		}
	}

	return n, err
}
