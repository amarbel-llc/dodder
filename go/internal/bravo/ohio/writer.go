package ohio

import (
	"encoding/binary"
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
)

func WriteAllOrDieTrying(w io.Writer, b []byte) (n int, err error) {
	var acc int

	for n < len(b) {
		acc, err = w.Write(b[n:])
		n += acc
		if err != nil {
			return n, err
		}
	}

	return n, err
}

func WriteInt8(w io.Writer, n int8) (written int, err error) {
	b := [1]byte{byte(n)}

	written, err = WriteAllOrDieTrying(w, b[:])
	if err != nil {
		err = errors.Wrap(err)
		return written, err
	}

	return written, err
}

func WriteUint8(w io.Writer, n uint8) (written int, err error) {
	b := [1]byte{n}

	written, err = WriteAllOrDieTrying(w, b[:])
	if err != nil {
		err = errors.Wrap(err)
		return written, err
	}

	return written, err
}

func WriteUint16(w io.Writer, n uint16) (written int, err error) {
	var intErr int
	var b [binary.MaxVarintLen16]byte

	intErr = binary.PutUvarint(b[:], uint64(n))

	if intErr != 1 {
		err = errors.ErrorWithStackf("expected to write %d but wrote %d", 2, intErr)
		return written, err
	}

	written, err = WriteAllOrDieTrying(w, b[:intErr])
	if err != nil {
		err = errors.Wrap(err)
		return written, err
	}

	return written, err
}

func WriteUint32(w io.Writer, n uint32) (written int, err error) {
	var intErr int
	var b [binary.MaxVarintLen32]byte

	intErr = binary.PutVarint(b[:], int64(n))

	if intErr != 1 {
		err = errors.ErrorWithStackf("expected to write %d but wrote %d", 2, intErr)
		return written, err
	}

	written, err = WriteAllOrDieTrying(w, b[:intErr])
	if err != nil {
		err = errors.Wrap(err)
		return written, err
	}

	return written, err
}

func WriteInt64(w io.Writer, n int64) (written int, err error) {
	var intErr int
	var b [binary.MaxVarintLen64]byte

	intErr = binary.PutVarint(b[:], n)

	written, err = WriteAllOrDieTrying(w, b[:intErr])
	if err != nil {
		err = errors.Wrap(err)
		return written, err
	}

	return written, err
}

func WriteFixedUInt16(w io.Writer, n uint16) (written int, err error) {
	b := UInt16ToByteArray(n)

	written, err = WriteAllOrDieTrying(w, b[:])
	if err != nil {
		err = errors.Wrap(err)
		return written, err
	}

	return written, err
}

func WriteFixedInt32(w io.Writer, n int32) (written int, err error) {
	b := Int32ToByteArray(n)

	written, err = WriteAllOrDieTrying(w, b[:])
	if err != nil {
		err = errors.Wrap(err)
		return written, err
	}

	return written, err
}

func WriteFixedInt64(w io.Writer, n int64) (written int, err error) {
	b := Int64ToByteArray(n)

	written, err = WriteAllOrDieTrying(w, b[:])
	if err != nil {
		err = errors.Wrap(err)
		return written, err
	}

	return written, err
}
