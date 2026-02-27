package toml

import (
	"io"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
)

type tomlBlobDecoder[
	BLOB any,
	BLOB_PTR interfaces.Ptr[BLOB],
] struct {
	blobWriterFactory domain_interfaces.BlobWriterFactory
	ignoreTomlErrors  bool
}

func MakeTomlBlobDecoderSaver[
	BLOB any,
	BLOB_PTR interfaces.Ptr[BLOB],
](
	blobWriter domain_interfaces.BlobWriterFactory,
) tomlBlobDecoder[BLOB, BLOB_PTR] {
	return tomlBlobDecoder[BLOB, BLOB_PTR]{
		blobWriterFactory: blobWriter,
	}
}

func MakeTomlDecoderIgnoreTomlErrors[
	BLOB any,
	BLOB_PTR interfaces.Ptr[BLOB],
](
	blobWriterFactory domain_interfaces.BlobWriterFactory,
) tomlBlobDecoder[BLOB, BLOB_PTR] {
	return tomlBlobDecoder[BLOB, BLOB_PTR]{
		blobWriterFactory: blobWriterFactory,
		ignoreTomlErrors:  true,
	}
}

func (decoder tomlBlobDecoder[BLOB, BLOB_PTR]) DecodeFrom(
	blob BLOB_PTR,
	reader io.Reader,
) (n int64, err error) {
	pipeReader, pipeWriter := io.Pipe()
	tomlDecoder := NewDecoder(pipeReader)

	chDone := make(chan error)

	go func(pr *io.PipeReader) {
		var err error
		defer func() {
			chDone <- err
			close(chDone)
		}()

		defer func() {
			if r := recover(); r != nil {
				if decoder.ignoreTomlErrors {
					err = nil
				} else {
					err = MakeError(errors.ErrorWithStackf("panicked during toml decoding: %s", r))
					pr.CloseWithError(errors.Wrap(err))
				}
			}
		}()

		if err = tomlDecoder.Decode(blob); err != nil {
			switch {
			case !errors.IsEOF(err) && !decoder.ignoreTomlErrors:
				err = errors.Wrap(MakeError(err))
				pr.CloseWithError(err)

			case !errors.IsEOF(err) && decoder.ignoreTomlErrors:
				err = nil
			}
		}

		ui.TodoP1("handle url parsing / validation")
	}(pipeReader)

	if n, err = io.Copy(pipeWriter, reader); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	if err = pipeWriter.Close(); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	if err = <-chDone; err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
