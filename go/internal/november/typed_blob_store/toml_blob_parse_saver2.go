package typed_blob_store

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"code.linenisgreat.com/dodder/go/lib/delta/toml"
)

type tomlBlobParseSaver2[
	O any,
] struct {
	awf              domain_interfaces.BlobWriterFactory
	ignoreTomlErrors bool
}

func MakeTomlBlobParseSaver2[
	O any,
](awf domain_interfaces.BlobWriterFactory,
) tomlBlobParseSaver2[O] {
	return tomlBlobParseSaver2[O]{
		awf: awf,
	}
}

func MakeTextParserIgnoreTomlErrors2[
	O any,
	OPtr interfaces.Ptr[O],
](awf domain_interfaces.BlobWriterFactory,
) tomlBlobParseSaver2[O] {
	return tomlBlobParseSaver2[O]{
		awf:              awf,
		ignoreTomlErrors: true,
	}
}

func (f tomlBlobParseSaver2[O]) DecodeFrom(
	t O,
	r io.Reader,
) (n int64, err error) {
	pr, pw := io.Pipe()
	td := toml.NewDecoder(pr)

	chDone := make(chan error)

	go func(pr *io.PipeReader) {
		var err error
		defer func() {
			chDone <- err
			close(chDone)
		}()

		defer func() {
			if r := recover(); r != nil {
				if f.ignoreTomlErrors {
					err = nil
				} else {
					err = toml.MakeError(errors.ErrorWithStackf("panicked during toml decoding: %s", r))
					pr.CloseWithError(errors.Wrap(err))
				}
			}
		}()

		if err = td.Decode(t); err != nil {
			switch {
			case !errors.IsEOF(err) && !f.ignoreTomlErrors:
				err = errors.Wrap(toml.MakeError(err))
				pr.CloseWithError(err)

			case !errors.IsEOF(err) && f.ignoreTomlErrors:
				err = nil
			}
		}

		ui.TodoP1("handle url parsing / validation")
	}(pr)

	if n, err = io.Copy(pw, r); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	if err = pw.Close(); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	if err = <-chDone; err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
