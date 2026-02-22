package mcp_madder

import "bytes"

type LimitingWriter struct {
	buf       bytes.Buffer
	maxBytes  int
	bytesSeen int
}

func MakeLimitingWriter(maxBytes int) *LimitingWriter {
	return &LimitingWriter{maxBytes: maxBytes}
}

func (w *LimitingWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	w.bytesSeen += n

	remaining := w.maxBytes - w.buf.Len()
	if remaining <= 0 {
		return n, nil
	}

	if len(p) > remaining {
		p = p[:remaining]
	}

	w.buf.Write(p)
	return n, nil
}

func (w *LimitingWriter) WriteString(s string) (n int, err error) {
	return w.Write([]byte(s))
}

func (w *LimitingWriter) String() string {
	return w.buf.String()
}

func (w *LimitingWriter) Truncated() bool {
	return w.bytesSeen > w.maxBytes
}

func (w *LimitingWriter) BytesSeen() int {
	return w.bytesSeen
}

func (w *LimitingWriter) BytesKept() int {
	return w.buf.Len()
}

func (w *LimitingWriter) Reset() {
	w.buf.Reset()
	w.bytesSeen = 0
}
