package toml

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/alfa/pool"
)

type TomlBlobEncoder[
	BLOB any,
	BLOB_PTR interfaces.Ptr[BLOB],
] struct{}

func (TomlBlobEncoder[BLOB, BLOB_PTR]) EncodeTo(
	blob BLOB_PTR,
	writer io.Writer,
) (n int64, err error) {
	bufferedWriter, repool := pool.GetBufferedWriter(writer)
	defer repool()

	defer errors.DeferredFlusher(&err, bufferedWriter)

	enc := NewEncoder(bufferedWriter)

	if err = enc.Encode(blob); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
