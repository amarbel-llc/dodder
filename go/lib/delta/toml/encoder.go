package toml

import (
	"io"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
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
