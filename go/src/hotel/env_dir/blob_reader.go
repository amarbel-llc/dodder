package env_dir

import (
	"bytes"
	"io"
	"os"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/markl_io"
	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
	"code.linenisgreat.com/dodder/go/src/charlie/files"
)

// TODO move into own package

type blobReader struct {
	readSeeker io.ReadSeeker
	digester   domain_interfaces.BlobWriter
	decrypter  io.Reader
	expander   io.ReadCloser
	tee        io.Reader
}

func NewReader(
	config Config,
	readSeeker io.ReadSeeker,
) (reader *blobReader, err error) {
	reader = &blobReader{
		readSeeker: readSeeker,
	}

	if reader.decrypter, err = config.GetBlobEncryption().WrapReader(
		reader.readSeeker,
	); err != nil {
		err = errors.Wrap(err)
		return reader, err
	}

	if reader.expander, err = config.GetBlobCompression().WrapReader(
		reader.decrypter,
	); err != nil {
		// TODO remove this when compression / encryption issues are resolved
		if _, err = reader.readSeeker.Seek(0, io.SeekStart); err != nil {
			err = errors.Wrap(err)
			return reader, err
		}

		if reader.expander, err = compression_type.CompressionTypeNone.WrapReader(
			reader.readSeeker,
		); err != nil {
			err = errors.Wrap(err)
			return reader, err
		}
	}

	hash, hashRepool := config.hashFormat.GetHash()
	defer hashRepool()
	reader.digester = markl_io.MakeWriter(hash, nil)
	reader.tee = io.TeeReader(reader.expander, reader.digester)

	return reader, err
}

func NewFileReaderOrErrNotExist(
	config Config,
	path string,
) (blobReader domain_interfaces.BlobReader, err error) {
	var readSeeker io.ReadSeeker

	if path == "-" {
		readSeeker = os.Stdin
	} else {
		var file *os.File

		if file, err = files.Open(path); err != nil {
			if !errors.IsNotExist(err) {
				err = errors.Wrap(err)
			}

			return blobReader, err
		}

		readSeeker = file
	}

	if blobReader, err = newFileReaderFromReadSeeker(
		config,
		readSeeker,
	); err != nil {
		err = errors.Wrap(err)
		return blobReader, err
	}

	return blobReader, err
}

func NewNopReader() (blobReader domain_interfaces.BlobReader, err error) {
	return newFileReaderFromReadSeeker(DefaultConfig, bytes.NewReader(nil))
}

func newFileReaderFromReadSeeker(
	config Config,
	readSeeker io.ReadSeeker,
) (blobReader domain_interfaces.BlobReader, err error) {
	// try the existing options. if they fail, try without encryption
	if blobReader, err = NewReader(
		config,
		readSeeker,
	); err != nil {
		if _, err = readSeeker.Seek(0, io.SeekStart); err != nil {
			err = errors.Wrap(err)
			return blobReader, err
		}

		config = MakeConfig(
			config.hashFormat,
			config.funcJoin,
			config.GetBlobCompression(),
			nil,
		)

		if blobReader, err = NewReader(
			config,
			readSeeker,
		); err != nil {
			err = errors.Wrap(err)
			return blobReader, err
		}
	}

	return blobReader, err
}

func (reader *blobReader) Seek(
	offset int64,
	whence int,
) (actual int64, err error) {
	seeker, ok := reader.decrypter.(io.Seeker)

	if !ok {
		err = errors.ErrorWithStackf("seeking not supported")
		return actual, err
	}

	return seeker.Seek(offset, whence)
}

func (reader *blobReader) ReadAt(p []byte, off int64) (n int, err error) {
	readerAt, ok := reader.decrypter.(io.ReaderAt)

	if !ok {
		err = errors.ErrorWithStackf(
			"reading at not supported for decryptor: %T",
			reader.decrypter,
		)

		return n, err
	}

	if n, err = readerAt.ReadAt(p, off); err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF)
		return n, err
	}

	return n, err
}

func (reader *blobReader) WriteTo(w io.Writer) (n int64, err error) {
	return io.Copy(w, reader.tee)
}

func (reader *blobReader) Read(p []byte) (n int, err error) {
	return reader.tee.Read(p)
}

func (reader *blobReader) Close() (err error) {
	if err = reader.expander.Close(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if closer, ok := reader.readSeeker.(io.Closer); ok {
		if err = closer.Close(); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (reader *blobReader) GetMarklId() domain_interfaces.MarklId {
	return reader.digester.GetMarklId()
}
