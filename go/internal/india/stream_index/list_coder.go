package stream_index

import (
	"bufio"
	"io"

	"code.linenisgreat.com/dodder/go/internal/_/key_bytes"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
)

type ListCoder struct {
	encoder binaryEncoder
	decoder binaryDecoder
}

func (coder *ListCoder) EncodeTo(
	object *sku.Transacted,
	writer *bufio.Writer,
) (n int64, err error) {
	if n, err = coder.encoder.writeFormat(
		writer,
		objectWithSigil{Transacted: object, Sigil: ids.SigilLatest},
	); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (coder *ListCoder) DecodeFrom(
	object *sku.Transacted,
	reader *bufio.Reader,
) (n int64, err error) {
	var n1 int
	var n2 int64

	coder.decoder.binaryField.Reset()
	coder.decoder.Buffer.Reset()

	n1, coder.decoder.ContentLength, err = ohio.ReadFixedUInt16(reader)
	n += int64(n1)

	if err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) && n == 0 {
			err = io.EOF
		}

		err = errors.WrapExceptSentinel(err, io.EOF)
		return n, err
	}

	contentLength64 := int64(coder.decoder.ContentLength)

	coder.decoder.limitedReader.R = reader
	coder.decoder.limitedReader.N = contentLength64

	// Read sigil field first (required by format)
	n2, err = coder.decoder.binaryField.ReadFrom(&coder.decoder.limitedReader)
	n += n2

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	// Consume the sigil but don't filter on it
	if coder.decoder.Key == key_bytes.Sigil {
		// sigil consumed, continue to remaining fields
	} else {
		// Not a sigil — treat as a regular field
		if err = coder.decoder.readFieldKey(object); err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	// Read remaining fields
	for coder.decoder.limitedReader.N > 0 {
		n2, err = coder.decoder.binaryField.ReadFrom(&coder.decoder.limitedReader)
		n += n2

		if err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		if err = coder.decoder.readFieldKey(object); err != nil {
			err = errors.Wrap(err)
			return n, err
		}
	}

	return n, err
}
