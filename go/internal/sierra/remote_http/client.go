package remote_http

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/genesis_configs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/inventory_list_coders"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/pool"
)

func MakeClient(
	envUI env_ui.Env,
	transport http.RoundTripper,
	repo *local_working_copy.Repo,
	typedBlobStore inventory_list_coders.Closet,
) *client {
	client := &client{
		envUI: envUI,
		http: http.Client{
			Transport: transport,
		},
		repo:                     repo,
		inventoryListCoderCloset: typedBlobStore,
	}

	client.Initialize()

	return client
}

type client struct {
	envUI                    env_ui.Env
	configImmutable          genesis_configs.TypedConfigPublic
	http                     http.Client
	repo                     *local_working_copy.Repo
	inventoryListCoderCloset inventory_list_coders.Closet
}

func (client *client) Initialize() {
	var request *http.Request

	{
		var err error

		if request, err = client.newRequest(
			"GET",
			"/config-immutable",
			nil,
		); err != nil {
			client.envUI.Cancel(err)
		}
	}

	var response *http.Response

	{
		var err error

		if response, err = client.http.Do(request); err != nil {
			errors.ContextCancelWithError(
				client.envUI,
				err,
			)
		}
	}

	if _, err := genesis_configs.CoderPublic.DecodeFrom(
		&client.configImmutable,
		response.Body,
	); err != nil {
		errors.ContextCancelWithErrorAndFormat(
			client.envUI,
			err,
			"failed to read remote immutable config",
		)
	}
}

func (client *client) GetEnv() env_ui.Env {
	return client.envUI
}

func (client *client) GetImmutableConfigPublic() genesis_configs.ConfigPublic {
	return client.configImmutable.Blob
}

func (client *client) GetImmutableConfigPublicType() ids.TypeStruct {
	return ids.TypeStruct(client.configImmutable.Type)
}

func (client *client) GetInventoryListStore() sku.InventoryListStore {
	return client
}

func (client *client) GetInventoryListCoderCloset() inventory_list_coders.Closet {
	return client.inventoryListCoderCloset
}

func (client *client) GetObjectStore() sku.RepoStore {
	return &httpRemoteObjectStore{client: client}
}

func (client *client) MakeImporter(
	options repo.ImporterOptions,
	storeOptions sku.StoreOptions,
) repo.Importer {
	panic(errors.Err501NotImplemented)
}

func (client *client) ImportSeq(
	seq sku.Seq,
	importer repo.Importer,
) (err error) {
	panic(errors.Err501NotImplemented)
}

func (client *client) MakeExternalQueryGroup(
	builderOptions queries.BuilderOption,
	externalQueryOptions sku.ExternalQueryOptions,
	args ...string,
) (qg *queries.Query, err error) {
	panic(errors.Err501NotImplemented)
}

func (client *client) MakeInventoryList(
	queryGroup *queries.Query,
) (list *sku.HeapTransacted, err error) {
	var request *http.Request
	listTypeString := client.GetImmutableConfigPublic().GetInventoryListTypeId()

	if request, err = client.newRequest(
		"GET",
		fmt.Sprintf(
			"/query/%s/%s",
			url.QueryEscape(listTypeString),
			url.QueryEscape(queryGroup.String()),
		),
		nil,
	); err != nil {
		err = errors.Wrap(err)
		return list, err
	}

	var response *http.Response

	if response, err = client.http.Do(request); err != nil {
		err = errors.ErrorWithStackf("failed to read response: %w", err)
		return list, err
	}

	if err = ReadErrorFromBodyOnNot(response, 200); err != nil {
		err = errors.Wrap(err)
		return list, err
	}

	inventoryListCoderCloset := client.repo.GetInventoryListCoderCloset()

	if list, err = inventoryListCoderCloset.ReadInventoryListBlob(
		client.repo.GetEnvRepo(),
		ids.GetOrPanic(listTypeString).TypeStruct,
		bufio.NewReader(response.Body),
	); err != nil {
		err = errors.Wrap(err)
		return list, err
	}

	return list, err
}

func (client *client) PullQueryGroupFromRemote(
	remote repo.Repo,
	queryGroup *queries.Query,
	options repo.ImporterOptions,
) (err error) {
	return client.pullQueryGroupFromWorkingCopy(
		remote.(repo.Repo),
		queryGroup,
		options,
	)
}

func (client *client) pullQueryGroupFromWorkingCopy(
	remote repo.Repo,
	queryGroup *queries.Query,
	options repo.ImporterOptions,
) (err error) {
	var list *sku.HeapTransacted

	if list, err = remote.MakeInventoryList(queryGroup); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// TODO local / remote version negotiation

	listType := ids.GetOrPanic(
		client.repo.GetImmutableConfigPublic().GetInventoryListTypeId(),
	).TypeStruct

	buffer := bytes.NewBuffer(nil)

	bufferedWriter, repoolBufferedWriter := pool.GetBufferedWriter(
		buffer,
	)
	defer repoolBufferedWriter()

	inventoryListCoderCloset := client.repo.GetInventoryListCoderCloset()

	for {
		errors.ContextContinueOrPanic(client.envUI)

		// TODO make a reader version of inventory lists to avoid allocation
		if _, err = inventoryListCoderCloset.WriteTypedBlobToWriterComputingBlobDigest(
			client.envUI,
			listType,
			client.repo.GetBlobStore().GetDefaultHashType(),
			quiter.MakeSeqErrorFromSeq(list.All()),
			bufferedWriter,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		var response *http.Response

		if err = bufferedWriter.Flush(); err != nil {
			err = errors.Wrap(err)
			return err
		}

		{
			var request *http.Request

			if request, err = client.newRequest(
				"POST",
				"/inventory_lists",
				buffer,
			); err != nil {
				err = errors.Wrap(err)
				return err
			}

			if options.AllowMergeConflicts {
				// TODO move to constant
				request.Header.Add(
					"x-dodder-remote_transfer_options-allow_merge_conflicts",
					"true",
				)
			}

			if response, err = client.http.Do(request); err != nil {
				err = errors.ErrorWithStackf("failed to read response: %w", err)
				return err
			}
		}

		if err = ReadErrorFromBodyOnNot(
			response,
			http.StatusCreated,
			http.StatusExpectationFailed,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		bufferedReader := bufio.NewReader(response.Body)

		var listMissingObjects *sku.HeapTransacted

		if listMissingObjects, err = client.inventoryListCoderCloset.ReadInventoryListBlob(
			client.GetEnv(),
			ids.GetOrPanic(
				client.configImmutable.Blob.GetInventoryListTypeId(),
			).TypeStruct,
			bufferedReader,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if err = response.Body.Close(); err != nil {
			err = errors.Wrap(err)
			return err
		}

		ui.Log().Print(
			"received missing blob list: %d",
			listMissingObjects.Len(),
		)

		for expected := range listMissingObjects.All() {
			ui.Err().Printf(
				"(requested) %q, sending blob",
				sku.String(expected),
			)

			errors.ContextContinueOrPanic(client.envUI)

			if err = client.WriteBlobToRemote(
				remote.GetBlobStore(),
				expected.GetBlobDigest(),
			); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}

		if response.StatusCode == http.StatusCreated {
			ui.Log().Print("done")
			return err
		}

		buffer.Reset()
		bufferedWriter.Reset(buffer)
	}
}

// ReadObjectHistory fetches the object's full version history from the server
// over the /object-history route. The parent negotiator needs the remote's
// complete history to find the merge base by TAI; the incremental transfer
// payload does not carry it (#299). Mirrors MakeInventoryList's decode.
func (client *client) ReadObjectHistory(
	oid *ids.ObjectId,
) (skus []*sku.Transacted, err error) {
	var request *http.Request

	if request, err = client.newRequest(
		"GET",
		fmt.Sprintf(
			"/object-history/%s",
			url.QueryEscape(oid.String()),
		),
		nil,
	); err != nil {
		err = errors.Wrap(err)
		return skus, err
	}

	var response *http.Response

	if response, err = client.http.Do(request); err != nil {
		err = errors.ErrorWithStackf("failed to read response: %w", err)
		return skus, err
	}

	if err = ReadErrorFromBodyOnNot(response, 200); err != nil {
		err = errors.Wrap(err)
		return skus, err
	}

	listTypeString := client.GetImmutableConfigPublic().GetInventoryListTypeId()
	inventoryListCoderCloset := client.repo.GetInventoryListCoderCloset()

	var list *sku.HeapTransacted

	if list, err = inventoryListCoderCloset.ReadInventoryListBlob(
		client.repo.GetEnvRepo(),
		ids.GetOrPanic(listTypeString).TypeStruct,
		bufio.NewReader(response.Body),
	); err != nil {
		err = errors.Wrap(err)
		return skus, err
	}

	for object := range list.All() {
		clone, _ := object.CloneTransacted() //repool:owned
		skus = append(skus, clone)
	}

	return skus, err
}

// configDescriptorJSON is the wire shape of the RFC-0005 GET /config
// response: the serving repo's current config-log head, naming a config
// TOML blob (fetched separately via the blob route) and its type. Shared
// by the server handler and the client fetch so the two stay in lockstep.
type configDescriptorJSON struct {
	BlobId     string `json:"blob-id"`
	ConfigType string `json:"config-type"`
	Tai        string `json:"tai,omitempty"`
}

// ConfigDescriptor is the parsed RFC-0005 config descriptor a clone over
// the HTTP backend fetches from the source to seed its config log. It is
// the empty zero value (BlobId == "") when the server offered none.
type ConfigDescriptor struct {
	BlobId     string
	ConfigType string
	Tai        string
}

// FetchConfigDescriptor issues GET /config against an HTTP remote and
// returns the source's config descriptor (RFC 0005 §HTTP Backend
// Transport). A 404 — or a server that does not route /config — means "no
// config offered" and yields the zero descriptor with offered == false and
// no error, so the clone keeps its genesis default. The remote MUST be the
// concrete HTTP client this package returns (a clone over the http backend
// always is); any other repo type yields offered == false.
func FetchConfigDescriptor(
	remote repo.Repo,
) (descriptor ConfigDescriptor, offered bool, err error) {
	httpClient, ok := remote.(*client)
	if !ok {
		return descriptor, false, err
	}

	var request *http.Request

	if request, err = httpClient.newRequest(
		"GET",
		"/config",
		nil,
	); err != nil {
		err = errors.Wrap(err)
		return descriptor, false, err
	}

	var response *http.Response

	if response, err = httpClient.http.Do(request); err != nil {
		err = errors.ErrorWithStackf("failed to read response: %w", err)
		return descriptor, false, err
	}

	defer errors.DeferredCloser(&err, response.Body)

	// A server predating RFC 0005 has no /config route (mux 404s the
	// unknown path) and a server with an empty config log returns 404
	// explicitly; both mean "no config offered".
	if response.StatusCode == http.StatusNotFound {
		return descriptor, false, err
	}

	if err = ReadErrorFromBodyOnNot(response, http.StatusOK); err != nil {
		err = errors.Wrap(err)
		return descriptor, false, err
	}

	var wire configDescriptorJSON

	if err = json.NewDecoder(response.Body).Decode(&wire); err != nil {
		err = errors.Wrapf(err, "decoding config descriptor")
		return descriptor, false, err
	}

	descriptor = ConfigDescriptor{
		BlobId:     wire.BlobId,
		ConfigType: wire.ConfigType,
		Tai:        wire.Tai,
	}

	return descriptor, true, err
}
