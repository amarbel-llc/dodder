package remote_http

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

// httpRemoteObjectStore satisfies sku.RepoStore over HTTP for the
// URL-transport client. Backs client.GetObjectStore() so edge
// expansion during pull can fetch parents over the wire.
//
// ReadPrimitiveQuery is left unimplemented (#172) — every existing
// caller passes nil and runs against a local store.
type httpRemoteObjectStore struct {
	client *client
}

// ReadOneInto issues GET /objects/{oid} and decodes the one-element
// inventory-list body into out. 404 maps to errors.MakeErrNotFound so
// edgeExplorer's IsErrNotFound branch keeps working.
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

	var fetched *sku.Transacted

	if fetched, err = store.client.repo.GetInventoryListCoderCloset().ReadInventoryListObject(
		store.client.repo.GetEnvRepo(),
		ids.GetOrPanic(listTypeString).TypeStruct,
		bufio.NewReader(response.Body),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if fetched == nil {
		// 200 with empty body — treat as not-found rather than
		// surfacing a successful-but-empty result.
		err = errors.MakeErrNotFound(oid)
		return err
	}

	sku.TransactedResetter.ResetWith(out, fetched)

	return err
}

// ReadPrimitiveQuery is deferred to #172. Every existing caller passes
// nil and runs against a local store; none reach the URL transport
// today.
func (store *httpRemoteObjectStore) ReadPrimitiveQuery(
	qg sku.PrimitiveQueryGroup,
	funcIter interfaces.FuncIter[*sku.Transacted],
) error {
	panic(errors.Wrapf(
		errors.Err501NotImplemented,
		"httpRemoteObjectStore.ReadPrimitiveQuery: see #172",
	))
}
