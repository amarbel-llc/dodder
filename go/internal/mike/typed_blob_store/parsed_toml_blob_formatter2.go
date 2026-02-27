package typed_blob_store

import (
	"bufio"
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/toml"
)

type ParsedBlobTomlFormatter2[O any] struct{}

func (ParsedBlobTomlFormatter2[O]) EncodeTo(
	t O,
	w1 io.Writer,
) (n int64, err error) {
	w := bufio.NewWriter(w1)
	defer errors.DeferredFlusher(&err, w)

	enc := toml.NewEncoder(w)

	if err = enc.Encode(t); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
