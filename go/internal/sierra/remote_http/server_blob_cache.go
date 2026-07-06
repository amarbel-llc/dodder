package remote_http

import (
	"sync"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"code.linenisgreat.com/dodder/go/lib/bravo/tridex"
	"github.com/amarbel-llc/madder/go/pkgs/fd"
	"github.com/amarbel-llc/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

type serverBlobCache struct {
	ui             fd.Std
	localBlobStore mad_domain_interfaces.BlobStore
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
	blobSha mad_domain_interfaces.MarklId,
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
