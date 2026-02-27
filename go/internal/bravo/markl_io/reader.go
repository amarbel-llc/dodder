package markl_io

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
)

type readCloser struct {
	tee    io.Reader
	reader io.Reader
	writer io.Writer
	hash   domain_interfaces.Hash
}

var _ domain_interfaces.BlobReader = readCloser{}

func MakeReadCloser(
	hash domain_interfaces.Hash,
	reader io.Reader,
) (src readCloser) {
	switch rt := reader.(type) {
	case *readCloser:
		src = *rt

	case readCloser:
		src = rt

	default:
		src.hash = hash
		src.reader = reader
	}

	src.setupTee()

	return src
}

func MakeReadCloserTee(
	hash domain_interfaces.Hash,
	reader io.Reader,
	writer io.Writer,
) (src readCloser) {
	switch rt := reader.(type) {
	case *readCloser:
		src = *rt
		src.writer = writer

	case readCloser:
		src = rt
		src.writer = writer

	default:
		src.hash = hash
		src.reader = reader
		src.writer = writer
	}

	src.setupTee()

	return src
}

func (readCloser *readCloser) setupTee() {
	if readCloser.writer == nil {
		readCloser.tee = io.TeeReader(
			readCloser.reader,
			readCloser.hash,
		)
	} else {
		readCloser.tee = io.TeeReader(
			readCloser.reader,
			io.MultiWriter(readCloser.hash, readCloser.writer),
		)
	}
}

func (readCloser readCloser) Seek(
	offset int64,
	whence int,
) (actual int64, err error) {
	seeker, ok := readCloser.reader.(io.Seeker)

	if !ok {
		err = errors.ErrorWithStackf("seeking not supported")
		return actual, err
	}

	return seeker.Seek(offset, whence)
}

func (readCloser readCloser) ReadAt(p []byte, off int64) (n int, err error) {
	readerAt, ok := readCloser.reader.(io.ReaderAt)

	if !ok {
		err = errors.ErrorWithStackf("reading at not supported")
		return n, err
	}

	return readerAt.ReadAt(p, off)
}

func (readCloser readCloser) WriteTo(w io.Writer) (n int64, err error) {
	// TODO-P3 determine why something in the copy returns an EOF
	if n, err = io.Copy(w, readCloser.tee); err != nil {
		if errors.IsEOF(err) {
			err = nil
		} else {
			err = errors.Wrap(err)
			return n, err
		}
	}

	return n, err
}

func (readCloser readCloser) Read(b []byte) (n int, err error) {
	return readCloser.tee.Read(b)
}

func (readCloser readCloser) Close() (err error) {
	if c, ok := readCloser.reader.(io.Closer); ok {
		defer errors.DeferredCloser(&err, c)
	}

	if c, ok := readCloser.writer.(io.Closer); ok {
		defer errors.DeferredCloser(&err, c)
	}

	return err
}

func (readCloser readCloser) GetMarklId() domain_interfaces.MarklId {
	digest, _ := readCloser.hash.GetMarklId()
	return digest
}

type nopReadCloser struct {
	hash domain_interfaces.Hash
	io.ReadCloser
}

var _ domain_interfaces.BlobReader = nopReadCloser{}

func MakeNopReadCloser(
	hash domain_interfaces.Hash,
	readCloser io.ReadCloser,
) domain_interfaces.BlobReader {
	return nopReadCloser{
		hash:       hash,
		ReadCloser: readCloser,
	}
}

func (nopReadCloser) Seek(offset int64, whence int) (actual int64, err error) {
	err = errors.ErrorWithStackf("seeking not supported")
	return actual, err
}

func (nopReadCloser) ReadAt(p []byte, off int64) (n int, err error) {
	err = errors.ErrorWithStackf("reading at not supported")
	return n, err
}

func (readCloser nopReadCloser) WriteTo(writer io.Writer) (n int64, err error) {
	return io.Copy(writer, readCloser.ReadCloser)
}

func (readCloser nopReadCloser) GetMarklId() domain_interfaces.MarklId {
	id, _ := readCloser.hash.GetMarklId()
	return id
}
