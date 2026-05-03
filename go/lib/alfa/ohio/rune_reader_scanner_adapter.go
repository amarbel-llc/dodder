package ohio

import "io"

type RuneReaderScannerAdapter struct {
	io.RuneReader
	Err error
}

func (adapter RuneReaderScannerAdapter) UnreadRune() error {
	return adapter.Err
}
