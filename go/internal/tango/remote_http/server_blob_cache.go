package remote_http

import (
	"sync"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/charlie/fd"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"code.linenisgreat.com/dodder/go/lib/delta/tridex"
)

type serverBlobCache struct {
	ui             fd.Std
	localBlobStore domain_interfaces.BlobStore
	shas           interfaces.TridexMutable
	init           sync.Once
}

func (serverBlobCache *serverBlobCache) populate() (err error) {
	serverBlobCache.shas = tridex.Make()

	{
		count := 0

		for sh, errIter := range serverBlobCache.localBlobStore.AllBlobs() {
			if errIter != nil {
				err = errors.Wrap(errIter)
				return err
			}

			serverBlobCache.shas.Add(markl.FormatBytesAsHex(sh))
			count++
		}

		ui.Log().Printf("have blobs: %d", count)
	}

	return err
}

func (serverBlobCache *serverBlobCache) HasBlob(
	blobSha domain_interfaces.MarklId,
) (ok bool, err error) {
	serverBlobCache.init.Do(
		func() {
			if err = serverBlobCache.populate(); err != nil {
				err = errors.Wrap(err)
			}
		},
	)

	if err != nil {
		return ok, err
	}

	if serverBlobCache.shas.ContainsExpansion(
		markl.FormatBytesAsHex(blobSha),
	) {
		ok = true
		return ok, err
	}

	return ok, err
}
