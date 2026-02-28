package remote_http

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"net/url"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/env_ui"
	"code.linenisgreat.com/dodder/go/internal/echo/genesis_configs"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/log_remote_inventory_lists"
	"code.linenisgreat.com/dodder/go/internal/india/inventory_list_coders"
	"code.linenisgreat.com/dodder/go/internal/kilo/queries"
	"code.linenisgreat.com/dodder/go/internal/quebec/repo"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/comments"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
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

	logRemoteInventoryLists log_remote_inventory_lists.Log
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

	client.logRemoteInventoryLists = log_remote_inventory_lists.Make(
		client.repo.GetEnvRepo(),
		client.repo.GetEnvRepo(),
	)
}

func (client *client) GetEnv() env_ui.Env {
	return client.envUI
}

func (client *client) GetImmutableConfigPublic() genesis_configs.ConfigPublic {
	return client.configImmutable.Blob
}

func (client *client) GetImmutableConfigPublicType() ids.TypeStruct {
	return client.configImmutable.Type
}

func (client *client) GetInventoryListStore() sku.InventoryListStore {
	return client
}

func (client *client) GetInventoryListCoderCloset() inventory_list_coders.Closet {
	return client.inventoryListCoderCloset
}

func (client *client) GetObjectStore() sku.RepoStore {
	return nil
}

func (client *client) MakeImporter(
	options repo.ImporterOptions,
	storeOptions sku.StoreOptions,
) repo.Importer {
	panic(comments.Implement())
}

func (client *client) ImportSeq(
	seq sku.Seq,
	importer repo.Importer,
) (err error) {
	return comments.Implement()
}

func (client *client) MakeExternalQueryGroup(
	builderOptions queries.BuilderOption,
	externalQueryOptions sku.ExternalQueryOptions,
	args ...string,
) (qg *queries.Query, err error) {
	err = comments.Implement()
	return qg, err
}

func (client *client) MakeInventoryList(
	queryGroup *queries.Query,
) (list *sku.HeapTransacted, err error) {
	var request *http.Request
	listTypeString := client.GetImmutableConfigPublic().GetInventoryListTypeId()

	if request, err = client.newRequest(
		"GET",
		fmt.Sprintf("/query/%s/%s",
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

// func (remoteHTTP *HTTP) PullQueryGroupFromRemote2(
// 	remote repo.ReadWrite,
// 	options repo.RemoteTransferOptions,
// 	queryStrings ...string,
// ) (err error) {
// 	var qg *query.Group

// 	if qg, err = remoteHTTP.MakeQueryGroup(queryStrings...); err != nil {
// 		err = errors.Wrap(err)
// 		return
// 	}

// 	if err = remoteHTTP.PullQueryGroupFromRemote(
// 		remote,
// 		qg,
// 		options,
// 	); err != nil {
// 		err = errors.Wrap(err)
// 		return
// 	}

// 	return
// }

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
		if _, err = inventoryListCoderCloset.WriteTypedBlobToWriter(
			client.envUI,
			listType,
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

func (client *client) ReadObjectHistory(
	oid *ids.ObjectId,
) (skus []*sku.Transacted, err error) {
	err = comments.Implement()
	return skus, err
}
