package triple_hyphen_io

import (
	"io"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/catgut"
)

const (
	Boundary = "---"
)

var BoundaryStringValue catgut.String

func init() {
	errors.PanicIfError(BoundaryStringValue.Set(Boundary))
}

var errBoundaryInvalid = newPkgError("boundary invalid")

type Peeker interface {
	io.Reader
	Peek(n int) (bytes []byte, err error)
}

func ReadBoundaryFromPeeker(peeker Peeker) (err error) {
	if err = PeekBoundaryFromPeeker(peeker); err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF, errBoundaryInvalid)
		return err
	}

	if _, err = io.CopyN(
		io.Discard,
		peeker,
		int64(BoundaryStringValue.Len()+1),
	); err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF)
		return err
	}

	return err
}

func PeekBoundaryFromPeeker(peeker Peeker) (err error) {
	var peeked []byte

	if peeked, err = peeker.Peek(BoundaryStringValue.Len() + 1); err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF)
		return err
	}

	boundaryOnly := peeked[:BoundaryStringValue.Len()]
	newline := peeked[BoundaryStringValue.Len()]

	if !BoundaryStringValue.EqualsBytes(boundaryOnly) {
		err = errBoundaryInvalid
		return err
	}

	if newline != '\n' {
		err = errBoundaryInvalid
		return err
	}

	return err
}

func ReadBoundaryFromRingBuffer(
	ringBuffer *catgut.RingBuffer,
) (err error) {
	var readable catgut.Slice

	readable, err = ringBuffer.PeekUpto('\n')

	if errors.IsNotNilAndNotEOF(err) {
		return err
	} else if readable.Len() == 0 && err == io.EOF {
		return err
	}

	if readable.Len() != BoundaryStringValue.Len() {
		err = errors.Wrapf(
			errBoundaryInvalid,
			"Expected: %d, Actual: %d. Readable: %q",
			BoundaryStringValue.Len(),
			readable.Len(),
			readable,
		)

		return err
	}

	if !readable.Equal(BoundaryStringValue.Bytes()) {
		err = errors.Wrapf(
			errBoundaryInvalid,
			"Expected: %q, Actual: %q",
			BoundaryStringValue.String(),
			readable.String(),
		)
		return err
	}

	// boundary and newline
	n := BoundaryStringValue.Len() + 1
	ringBuffer.AdvanceRead(n)

	if err == io.EOF {
		err = nil
	}

	return err
}
