package env_ui

import (
	"sync/atomic"

	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
)

type ProgressWriter struct {
	written atomic.Int64
}

func (writer *ProgressWriter) GetWrittenHumanString() string {
	written := writer.GetWritten()
	return ui.GetHumanBytesStringOrError(written)
}

func (writer *ProgressWriter) GetWritten() int64 {
	return writer.written.Load()
}

func (writer *ProgressWriter) Reset() {
	writer.written.Swap(0)
}

func (writer *ProgressWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	writer.written.Add(int64(n))
	return n, err
}
