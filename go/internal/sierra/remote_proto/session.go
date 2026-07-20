package remote_proto

import (
	"bufio"
	"io"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"github.com/DataDog/zstd"
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

// writeBlob writes a blob_header control frame (naming the blob and, when
// compressed, the algorithm), streams the blob's bytes — optionally through a
// zstd encoder — as a sequence of blob frames terminated by a zero-length
// frame, then flushes. The blob is never held in memory in full.
func (s *session) writeBlob(
	blobIdString string,
	reader io.Reader,
	compression string,
) (err error) {
	if err = writeControlFrame(s.writer, TypeBlobHeader, control{
		BlobId:      blobIdString,
		Compression: compression,
	}); err != nil {
		err = errors.Wrap(err)
		return err
	}

	frameWriter := &blobFrameWriter{session: s}

	switch compression {
	case CompressionZstd:
		encoder := zstd.NewWriter(frameWriter)

		if _, err = io.Copy(encoder, reader); err != nil {
			_ = encoder.Close()
			err = errors.Wrapf(err, "compressing blob %s", blobIdString)
			return err
		}

		if err = encoder.Close(); err != nil {
			err = errors.Wrapf(err, "finishing blob %s compression", blobIdString)
			return err
		}

	default:
		if _, err = io.Copy(frameWriter, reader); err != nil {
			err = errors.Wrapf(err, "reading blob %s", blobIdString)
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

// blobFrameWriter chops every Write into blob frames no larger than
// blobChunkSize. It is the sink for the raw or zstd-compressed blob byte
// stream.
type blobFrameWriter struct {
	session *session
}

func (w *blobFrameWriter) Write(p []byte) (n int, err error) {
	for len(p) > 0 {
		chunk := p
		if len(chunk) > blobChunkSize {
			chunk = chunk[:blobChunkSize]
		}

		if err = writeFrame(w.session.writer, frameKindBlob, chunk); err != nil {
			err = errors.Wrap(err)
			return n, err
		}

		n += len(chunk)
		p = p[len(chunk):]
	}

	return n, err
}

// blobFrameReader presents the blob frames between a blob_header and the
// zero-length terminator as a single io.Reader, so the receiver can stream
// them straight into the blob writer (or through a zstd decoder).
type blobFrameReader struct {
	session   *session
	remaining int
	done      bool
}

func (r *blobFrameReader) Read(p []byte) (n int, err error) {
	if r.done {
		return 0, io.EOF
	}

	if r.remaining == 0 {
		kind, length, frameErr := readFrameHeader(r.session.reader)
		if frameErr != nil {
			if errors.IsEOF(frameErr) {
				return 0, io.EOF
			}

			return 0, errors.Wrap(frameErr)
		}

		if kind != frameKindBlob {
			return 0, errors.Errorf(
				"expected blob frame, got kind %d",
				kind,
			)
		}

		if length == 0 {
			r.done = true
			return 0, io.EOF
		}

		r.remaining = int(length)
	}

	toRead := len(p)
	if toRead > r.remaining {
		toRead = r.remaining
	}

	n, err = r.session.reader.Read(p[:toRead])
	r.remaining -= n

	// Read errors other than a clean intra-frame boundary are real; a
	// truncated frame surfaces here.
	return n, err
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
