package remote_proto

import (
	"encoding/binary"
	"io"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

// frameHeaderLen is the fixed framing overhead: one kind byte plus a
// big-endian uint32 length.
const frameHeaderLen = 1 + 4

// writeFrame writes one frame — kind byte, big-endian uint32 length, then
// payload — to w.
func writeFrame(w io.Writer, kind frameKind, payload []byte) (err error) {
	var header [frameHeaderLen]byte
	header[0] = byte(kind)
	binary.BigEndian.PutUint32(header[1:], uint32(len(payload)))

	if _, err = w.Write(header[:]); err != nil {
		err = errors.Wrapf(err, "writing frame header")
		return err
	}

	if _, err = w.Write(payload); err != nil {
		err = errors.Wrapf(err, "writing frame payload")
		return err
	}

	return err
}

// readFrameHeader reads a frame's kind and payload length. The caller then
// reads exactly length bytes of payload (directly, for streaming, or via
// readFramePayload for buffered control frames).
func readFrameHeader(r io.Reader) (kind frameKind, length uint32, err error) {
	var header [frameHeaderLen]byte

	if _, err = io.ReadFull(r, header[:]); err != nil {
		// EOF here is the ordinary "peer closed the stream" signal; the
		// caller distinguishes it from a truncated payload.
		if !errors.IsEOF(err) {
			err = errors.Wrapf(err, "reading frame header")
		}
		return kind, length, err
	}

	kind = frameKind(header[0])
	length = binary.BigEndian.Uint32(header[1:])

	return kind, length, err
}

// readFramePayload reads exactly length bytes of a control or objects frame
// payload, rejecting an oversized length before allocating.
func readFramePayload(r io.Reader, length uint32) (payload []byte, err error) {
	if length > maxControlFrameLen {
		err = errors.Errorf(
			"frame length %d exceeds maximum %d",
			length,
			maxControlFrameLen,
		)
		return payload, err
	}

	payload = make([]byte, length)

	if _, err = io.ReadFull(r, payload); err != nil {
		err = errors.Wrapf(err, "reading frame payload (%d bytes)", length)
		return payload, err
	}

	return payload, err
}

// writeControlFrame encodes a control message and writes it as a single
// control frame.
func writeControlFrame(w io.Writer, typeString string, msg control) (err error) {
	var payload []byte

	if payload, err = encodeControl(typeString, msg); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = writeFrame(w, frameKindControl, payload); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
