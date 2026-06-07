package remote_proto

import (
	"bufio"
	"io"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

// session wraps the byte stream of one transfer with buffered framing. It
// is transport-agnostic: the underlying io.ReadWriteCloser may be a stdio
// pipe, a unix/TCP net.Conn, a websocket adapted via websocket.NetConn, or
// an in-memory net.Pipe (used by the unit tests).
type session struct {
	conn   io.ReadWriteCloser
	reader *bufio.Reader
	writer *bufio.Writer
}

func makeSession(conn io.ReadWriteCloser) *session {
	return &session{
		conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}
}

func (s *session) Close() error {
	return s.conn.Close()
}

// writeControl writes a control message as one control frame and flushes.
func (s *session) writeControl(typeString string, msg control) (err error) {
	if err = writeControlFrame(s.writer, typeString, msg); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = s.writer.Flush(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// writeObjects writes an encoded inventory_list payload as one objects
// frame and flushes.
func (s *session) writeObjects(payload []byte) (err error) {
	if err = writeFrame(s.writer, frameKindObjects, payload); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = s.writer.Flush(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// blobChunkSize bounds each blob frame so a blob of any size streams with
// constant memory rather than being buffered whole.
const blobChunkSize = 64 * 1024

// writeBlob writes a blob_header control frame, then streams the blob's bytes
// as a sequence of blob frames, terminated by a zero-length blob frame, then
// flushes. The blob is never held in memory in full.
func (s *session) writeBlob(blobIdString string, reader io.Reader) (err error) {
	if err = writeControlFrame(s.writer, TypeBlobHeader, control{
		BlobId: blobIdString,
	}); err != nil {
		err = errors.Wrap(err)
		return err
	}

	buffer := make([]byte, blobChunkSize)

	for {
		n, readErr := reader.Read(buffer)

		if n > 0 {
			if err = writeFrame(s.writer, frameKindBlob, buffer[:n]); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}

		if readErr != nil {
			if errors.IsEOF(readErr) {
				break
			}

			err = errors.Wrapf(readErr, "reading blob %s", blobIdString)
			return err
		}
	}

	// Zero-length terminator frame marks the end of the blob.
	if err = writeFrame(s.writer, frameKindBlob, nil); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = s.writer.Flush(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// readControl reads the next frame, which MUST be a control frame, and
// decodes it. Returns the type string (with leading "!") and the envelope.
func (s *session) readControl() (typeString string, msg control, err error) {
	kind, length, err := readFrameHeader(s.reader)
	if err != nil {
		// Propagate EOF unwrapped so callers can detect a clean close.
		if errors.IsEOF(err) {
			return typeString, msg, io.EOF
		}
		err = errors.Wrap(err)
		return typeString, msg, err
	}

	if kind != frameKindControl {
		err = errors.Errorf("expected control frame, got kind %d", kind)
		return typeString, msg, err
	}

	var payload []byte

	if payload, err = readFramePayload(s.reader, length); err != nil {
		err = errors.Wrap(err)
		return typeString, msg, err
	}

	if typeString, msg, err = decodeControl(payload); err != nil {
		err = errors.Wrap(err)
		return typeString, msg, err
	}

	return typeString, msg, err
}

// readControlExpecting reads a control frame and asserts its type. A
// TypeError frame is surfaced as the remote's error regardless of the
// expected type.
func (s *session) readControlExpecting(
	expectedTypeString string,
) (msg control, err error) {
	var typeString string

	if typeString, msg, err = s.readControl(); err != nil {
		// Never wrap a bare io.EOF (the errors package panics on it);
		// surface a descriptive terminal error instead.
		if errors.IsEOF(err) {
			err = errors.Errorf(
				"connection closed before %q frame",
				expectedTypeString,
			)
			return msg, err
		}

		err = errors.Wrap(err)
		return msg, err
	}

	if typeString == "!"+TypeError {
		err = errors.Errorf("remote error: %s", msg.Message)
		return msg, err
	}

	if typeString != "!"+expectedTypeString {
		err = errors.Errorf(
			"expected %q frame, got %q",
			expectedTypeString,
			typeString,
		)
		return msg, err
	}

	return msg, err
}

// writeError writes a terminal error control frame. The write error (if
// any) is best-effort; the caller's original error is what matters.
func (s *session) writeError(message string, statusCode int) {
	_ = s.writeControl(TypeError, control{
		Message:    message,
		StatusCode: statusCode,
	})
}
