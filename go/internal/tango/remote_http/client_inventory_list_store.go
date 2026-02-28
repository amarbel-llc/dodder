package remote_http

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/log_remote_inventory_lists"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/comments"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

func (client client) WriteInventoryListObject(t *sku.Transacted) (err error) {
	return comments.Implement()
}

// TODO add progress bar
func (client client) ImportInventoryList(
	blobStore domain_interfaces.BlobStore,
	listSku *sku.Transacted,
) (err error) {
	logEntry := log_remote_inventory_lists.Entry{
		EntryType:  log_remote_inventory_lists.EntryTypeSent,
		PublicKey:  client.configImmutable.Blob.GetPublicKey(),
		Transacted: listSku,
	}

	if err = client.logRemoteInventoryLists.Exists(
		logEntry,
	); errors.IsErrNotFound(err) {
		err = nil
	} else if err != nil {
		err = errors.Wrap(err)
		return err
	} else {
		return err
	}

	ui.Log().Printf("importing list: %s", sku.String(listSku))

	var sbListSkuBox strings.Builder

	{
		bufferedWriter, repoolBufferedWriter := pool.GetBufferedWriter(
			&sbListSkuBox,
		)
		defer repoolBufferedWriter()

		if _, err = client.inventoryListCoderCloset.WriteObjectToWriter(
			listSku.GetType().ToType(),
			listSku,
			bufferedWriter,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if err = bufferedWriter.Flush(); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	var listBlobReader io.ReadCloser

	if listBlobReader, err = blobStore.MakeBlobReader(listSku.GetBlobDigest()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var request *http.Request

	if request, err = client.newRequest(
		"POST",
		fmt.Sprintf(
			"/inventory_lists/%s/%s",
			listSku.GetType(),
			strings.TrimSpace(sbListSkuBox.String()),
		),
		listBlobReader,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	{
		pubKey := client.repo.GetImmutableConfigPublic().GetPublicKey()

		request.Header.Add(
			headerRepoPublicKey,
			pubKey.StringWithFormat(),
		)
	}

	{
		sec := client.repo.GetImmutableConfigPrivate().Blob.GetPrivateKey()

		var sig markl.Id

		if err = sec.Sign(
			listSku.GetBlobDigest(),
			&sig,
			markl.PurposeRequestRepoSigV1,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		request.Header.Add(headerRepoSig, sig.StringWithFormat())
	}

	// TODO ensure that conflicts were addressed prior to importing
	// if options.AllowMergeConflicts {
	// 	request.Header.Add("x-dodder-remote_transfer_options-allow_merge_conflicts",
	// "true")
	// }

	var response *http.Response

	if response, err = client.http.Do(request); err != nil {
		err = errors.ErrorWithStackf("failed to read response: %w", err)
		return err
	}

	if err = ReadErrorFromBodyOnNot(
		response,
		http.StatusCreated,
		http.StatusFound,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var digests markl.Slice

	if _, err = digests.ReadFrom(bufio.NewReader(response.Body)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if len(digests) > 0 {
		ui.Err().Printf("sending blobs: %d", len(digests))
	}

	for _, digest := range digests {
		if err = client.WriteBlobToRemote(blobStore, digest); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if err = response.Body.Close(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = client.logRemoteInventoryLists.Append(
		logEntry,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (client client) ReadLast() (max *sku.Transacted, err error) {
	return nil, comments.Implement()
}

func (client client) AllInventoryListContents(
	blobSha domain_interfaces.MarklId,
) interfaces.SeqError[*sku.Transacted] {
	return nil
}

func (client client) ReadAllSkus(
	f func(besty, sk *sku.Transacted) error,
) (err error) {
	return comments.Implement()
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
