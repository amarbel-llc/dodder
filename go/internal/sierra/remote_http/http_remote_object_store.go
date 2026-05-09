package remote_http

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

// httpRemoteObjectStore satisfies sku.RepoStore over HTTP, fronting
// the URL-transport client for callers that need single-object lookup
// (edge expansion during pull). Backs client.GetObjectStore(); without
// it that method returned nil and the URL clone path crashed in
// expandEdges (#171).
//
// ReadPrimitiveQuery is left unimplemented — no caller exercises it
// over the URL transport today (#172). When one appears, the wire
// shape ("scan all transacted") is small and additive.
type httpRemoteObjectStore struct {
	client *client
}

// ReadOneInto issues GET /objects/{oid} and decodes the one-element
// inventory-list response into out. 404 maps to errors.MakeErrNotFound
// so edgeExplorer's IsErrNotFound branch keeps working.
func (store *httpRemoteObjectStore) ReadOneInto(
	oid domain_interfaces.ObjectId,
	out *sku.Transacted,
) (err error) {
	var request *http.Request

	if request, err = store.client.newRequest(
		"GET",
		fmt.Sprintf("/objects/%s", url.QueryEscape(oid.String())),
		nil,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var response *http.Response

	if response, err = store.client.http.Do(request); err != nil {
		err = errors.ErrorWithStackf("failed to read response: %w", err)
		return err
	}

	defer errors.DeferredCloser(&err, response.Body)

	if response.StatusCode == http.StatusNotFound {
		err = errors.MakeErrNotFound(oid)
		return err
	}

	if err = ReadErrorFromBodyOnNot(response, http.StatusOK); err != nil {
		err = errors.Wrap(err)
		return err
	}

	listTypeString := store.client.GetImmutableConfigPublic().GetInventoryListTypeId()

	inventoryListCoderCloset := store.client.repo.GetInventoryListCoderCloset()

	var list *sku.HeapTransacted

	if list, err = inventoryListCoderCloset.ReadInventoryListBlob(
		store.client.repo.GetEnvRepo(),
		ids.GetOrPanic(listTypeString).TypeStruct,
		bufio.NewReader(response.Body),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if list.Len() == 0 {
		// Server returned 200 with an empty list. Treat as not-found
		// so callers don't see a successful-but-empty result.
		err = errors.MakeErrNotFound(oid)
		return err
	}

	first, ok := list.Peek()
	if !ok {
		err = errors.MakeErrNotFound(oid)
		return err
	}

	sku.TransactedResetter.ResetWith(out, first)

	return err
}

// ReadPrimitiveQuery is deferred to #172. Every existing caller of
// ReadPrimitiveQuery passes nil ("scan all"), but none reach the HTTP
// client today — they all run against in-process local stores. When
// one appears, the wire shape is straightforward (a streaming "all
// objects" endpoint). Until then, panic with a clear pointer.
func (store *httpRemoteObjectStore) ReadPrimitiveQuery(
	qg sku.PrimitiveQueryGroup,
	funcIter interfaces.FuncIter[*sku.Transacted],
) error {
	panic(errors.Wrapf(
		errors.Err501NotImplemented,
		"httpRemoteObjectStore.ReadPrimitiveQuery: see #172",
	))
}
