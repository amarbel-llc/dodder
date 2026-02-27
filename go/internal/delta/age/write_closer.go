package age

import "io"

// TODO-P3 remove
type writeCloser struct {
	io.Writer
}

func (w writeCloser) Close() (err error) {
	return err
}
