package remote_http

import (
	"net/http"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

func (client client) WriteInventoryListObject(t *sku.Transacted) (err error) {
	panic(errors.Err501NotImplemented)
}

func (client client) ReadLast() (max *sku.Transacted, err error) {
	panic(errors.Err501NotImplemented)
}

func (client client) AllInventoryListContents(
	blobSha domain_interfaces.MarklId,
) interfaces.SeqError[*sku.Transacted] {
	return nil
}

func (client client) ReadAllSkus(
	f func(besty, sk *sku.Transacted) error,
) (err error) {
	panic(errors.Err501NotImplemented)
}

func (client client) AllInventoryLists() interfaces.SeqError[*sku.Transacted] {
	var request *http.Request

	{
		var err error

		if request, err = client.newRequest(
			"GET",
			"/inventory_lists",
			nil,
		); err != nil {
			client.envUI.Cancel(err)
			return nil
		}
	}

	var response *http.Response

	{
		var err error

		if response, err = client.http.Do(request); err != nil {
			errors.ContextCancelWithErrorAndFormat(
				client.envUI,
				err,
				"failed to read response",
			)
			return nil
		}
	}

	return client.inventoryListCoderCloset.AllDecodedObjectsFromStream(
		response.Body,
		nil,
	)
}
