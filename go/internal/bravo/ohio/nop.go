package ohio

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
)

// TODO move this file into src/charlie/ohio_nop

func NopCloser(reader io.Reader) io.ReadCloser {
	switch reader.(type) {
	case interface {
		io.WriterTo
		io.ReaderAt
	}:
		return nopCloserReaderAtWriterTo{reader}

	case io.WriterTo:
		return nopCloserWriterTo{reader}

	default:
		return nopCloser{reader}
	}
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

type nopCloserWriterTo struct {
	io.Reader
}

func (nopCloserWriterTo) Close() error { return nil }

func (wrapper nopCloserWriterTo) WriteTo(w io.Writer) (n int64, err error) {
	return wrapper.Reader.(io.WriterTo).WriteTo(w)
}

type nopCloserReaderAtWriterTo struct {
	io.Reader
}

func (nopCloserReaderAtWriterTo) Close() error { return nil }

func (wrapper nopCloserReaderAtWriterTo) WriteTo(
	writer io.Writer,
) (n int64, err error) {
	return wrapper.Reader.(io.WriterTo).WriteTo(writer)
}

func (wrapper nopCloserReaderAtWriterTo) ReadAt(
	p []byte,
	off int64,
) (n int, err error) {
	readerAt := wrapper.Reader.(io.ReaderAt)

	if n, err = readerAt.ReadAt(p, off); err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF)
		return n, err
	}

	return n, err
}

type NopeIOWrapper struct{}

func (NopeIOWrapper) WrapReader(
	readerIn io.Reader,
) (readerOut io.ReadCloser, err error) {
	return NopCloser(readerIn), nil
}

func (NopeIOWrapper) WrapWriter(
	writerIn io.Writer,
) (io.WriteCloser, error) {
	return NopWriteCloser(writerIn), nil
}

type nopWriteCloser struct {
	io.Writer
}

func NopWriteCloser(writer io.Writer) nopWriteCloser {
	return nopWriteCloser{Writer: writer}
}

func (nopWriteCloser) Close() error {
	return nil
}
