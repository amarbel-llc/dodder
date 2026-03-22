package hyphence

import (
	"bufio"
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
)

type Decoder[BLOB any] struct {
	RequireMetadata      bool
	AllowMissingSeparator bool
	Metadata, Blob       interfaces.DecoderFromBufferedReader[BLOB]
	BlobTeeWriter        io.Writer
}

func (decoder *Decoder[BLOB]) DecodeFrom(
	blob BLOB,
	bufferedReader *bufio.Reader,
) (n int64, err error) {
	var n1 int64
	n1, err = decoder.readMetadataFrom(blob, bufferedReader)
	n += n1

	if err != nil {
		err = errors.Wrapf(err, "metadata read failed")
		return n, err
	}

	blobReader := bufferedReader

	if decoder.BlobTeeWriter != nil {
		blobReader = bufio.NewReader(
			io.TeeReader(bufferedReader, decoder.BlobTeeWriter),
		)
	}

	n1, err = decoder.Blob.DecodeFrom(blob, blobReader)
	n += n1

	if err != nil {
		err = errors.Wrapf(err, "blob read failed")
		return n, err
	}

	return n, err
}

func (decoder *Decoder[BLOB]) readMetadataFrom(
	blob BLOB,
	bufferedReader *bufio.Reader,
) (n int64, err error) {
	var state readerState

	if decoder.RequireMetadata && decoder.Metadata == nil {
		err = errors.ErrorWithStackf("metadata reader is nil")
		return n, err
	}

	if decoder.Blob == nil {
		err = errors.ErrorWithStackf("blob reader is nil")
		return n, err
	}

	var metadataPipe ohio.PipedReader

	{
		var isBoundary bool

		if err = ReadBoundaryFromPeeker(bufferedReader); err != nil {
			if err == errBoundaryInvalid {
				err = nil
			} else {
				err = errors.Wrap(err)
				return n, err
			}
		} else {
			isBoundary = true
		}

		switch {
		case decoder.RequireMetadata && !isBoundary:
			// TODO add context
			err = errors.Wrap(errBoundaryInvalid)
			return n, err

		case !isBoundary:
			state = readerStateSecondBoundary

		default:
			state = readerStateFirstBoundary
			metadataPipe = ohio.MakePipedDecoder(blob, decoder.Metadata)
		}
	}

	var isEOF bool

LINE_READ_LOOP:
	for !isEOF {
		var rawLine, line string

		rawLine, err = bufferedReader.ReadString('\n')
		n += int64(len(rawLine))

		if err == io.EOF {
			err = nil
			isEOF = true
		} else if err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		line = strings.TrimSuffix(rawLine, "\n")

		switch state {
		case readerStateEmpty:
			// nop, processing done above

		case readerStateFirstBoundary:
			if line == Boundary {
				if _, err = metadataPipe.Close(); err != nil {
					err = errors.Wrapf(err, "metadata read failed")
					return n, err
				}

				if err = decoder.peekSeparatorLine(bufferedReader); err != nil {
					return n, err
				}

				break LINE_READ_LOOP
			}

			if _, err = metadataPipe.Write([]byte(rawLine)); err != nil {
				err = errors.Wrap(err)
				return n, err
			}

		case readerStateSecondBoundary:
			// No opening boundary found — all content is blob
			break LINE_READ_LOOP

		default:
			err = errors.ErrorWithStackf("impossible state %d", state)
			return n, err
		}
	}

	return n, err
}

func (decoder *Decoder[BLOB]) peekSeparatorLine(
	bufferedReader *bufio.Reader,
) (err error) {
	peeked, err := bufferedReader.Peek(1)

	if err == io.EOF {
		// No blob content — that's fine
		return nil
	}

	if err != nil {
		return errors.Wrap(err)
	}

	if peeked[0] == '\n' {
		// Blank separator line — consume it and continue to blob
		if _, err = bufferedReader.ReadByte(); err != nil {
			return errors.Wrap(err)
		}

		return nil
	}

	if decoder.AllowMissingSeparator {
		// Non-strict: leave the line unconsumed for the blob reader
		return nil
	}

	return errors.Wrap(errMissingNewlineAfterBoundary)
}
