package remote_http

import (
	"bufio"
	"bytes"
	"io"
	"net/http"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/bravo/ohio"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/hotel/env_dir"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/tango/repo"
	"code.linenisgreat.com/dodder/go/src/victor/local_working_copy"
)

func (server *Server) writeInventoryListTypedBlobLocalWorkingCopy(
	local *local_working_copy.Repo,
	request Request,
) (response Response) {
	listCoderCloset := server.Repo.GetInventoryListCoderCloset()

	responseBuffer := bytes.NewBuffer(nil)

	// TODO make option to read from headers
	// TODO add remote blob store
	importerOptions := repo.ImporterOptions{
		// TODO
		CheckedOutPrinter: local.PrinterCheckedOutConflictsForRemoteTransfers(),
	}

	if request.Headers.Get(
		"x-dodder-remote_transfer_options-allow_merge_conflicts",
	) == "true" {
		importerOptions.AllowMergeConflicts = true
	}

	listMissingObjects := sku.MakeListTransacted()
	var requestRetry bool

	importerOptions.BlobCopierDelegate = func(
		result sku.BlobCopyResult,
	) (err error) {
		errors.ContextContinueOrPanic(server.Repo.GetEnv())

		if !result.IsMissing() {
			return err
		}

		if result.ObjectOrNil.GetGenre() == genres.InventoryList {
			requestRetry = true
		}

		ui.Log().Print(
			"missing blob for list: %s",
			sku.String(result.ObjectOrNil),
		)

		clonedObj, _ := result.ObjectOrNil.CloneTransacted()
		listMissingObjects.Add(clonedObj)

		return err
	}

	importer := server.Repo.MakeImporter(
		importerOptions,
		sku.GetStoreOptionsRemoteTransfer(),
	)

	seq := listCoderCloset.AllDecodedObjectsFromStream(
		bufio.NewReader(request.Body),
		nil,
	)

	if err := server.Repo.ImportSeq(
		seq,
		importer,
	); err != nil {
		if env_dir.IsErrBlobMissing(err) {
			requestRetry = true
		} else {
			response.Error(err)
			return response
		}
	}

	bufferedWriter, repoolBufferedWriter := pool.GetBufferedWriter(
		responseBuffer,
	)
	defer repoolBufferedWriter()

	listType := ids.GetOrPanic(
		local.GetImmutableConfigPublic().GetInventoryListTypeId(),
	).TypeStruct

	inventoryListCoderCloset := server.Repo.GetInventoryListCoderCloset()

	if _, err := inventoryListCoderCloset.WriteBlobToWriter(
		local,
		listType,
		quiter.MakeSeqErrorFromSeq(listMissingObjects.All()),
		bufferedWriter,
	); err != nil {
		response.Error(err)
		return response
	}

	if err := bufferedWriter.Flush(); err != nil {
		response.Error(err)
		return response
	}

	if requestRetry {
		response.StatusCode = http.StatusExpectationFailed
	} else {
		response.StatusCode = http.StatusCreated
	}

	response.Body = ohio.NopCloser(responseBuffer)

	return response
}

func (server *Server) writeInventoryListLocalWorkingCopy(
	local *local_working_copy.Repo,
	request Request,
	listSku *sku.Transacted,
) (response Response) {
	listSkuType := ids.GetOrPanic(
		server.Repo.GetImmutableConfigPublic().GetInventoryListTypeId(),
	).TypeStruct

	blobStore := server.Repo.GetBlobStore()

	if listSku != nil {
		if listSku.GetGenre() != genres.InventoryList {
			response.Error(genres.MakeErrUnsupportedGenre(listSku.GetGenre()))
			return response
		}

		if blobStore.HasBlob(listSku.GetBlobDigest()) {
			response.StatusCode = http.StatusFound
			return response
		}

		listSkuType = listSku.GetType().ToType()
	}

	listCoderCloset := server.Repo.GetInventoryListCoderCloset()

	var blobWriter domain_interfaces.BlobWriter

	{
		var err error

		if blobWriter, err = blobStore.MakeBlobWriter(nil); err != nil {
			response.Error(err)
			return response
		}
	}

	var list *sku.HeapTransacted

	{
		var err error

		if list, err = listCoderCloset.ReadInventoryListBlob(
			local,
			listSkuType,
			bufio.NewReader(io.TeeReader(request.Body, blobWriter)),
		); err != nil {
			response.Error(err)
			return response
		}
	}

	ui.Log().Printf("read list: %d objects", list.Len())

	responseBuffer := bytes.NewBuffer(nil)

	// TODO make option to read from headers
	// TODO add remote blob store
	importerOptions := repo.ImporterOptions{
		// TODO
		CheckedOutPrinter: local.PrinterCheckedOutConflictsForRemoteTransfers(),
	}

	if request.Headers.Get(
		"x-dodder-remote_transfer_options-allow_merge_conflicts",
	) == "true" {
		importerOptions.AllowMergeConflicts = true
	}

	listMissingSkus := sku.MakeListTransacted()
	var requestRetry bool

	importerOptions.BlobCopierDelegate = func(
		result sku.BlobCopyResult,
	) (err error) {
		errors.ContextContinueOrPanic(server.Repo.GetEnv())

		ui.Debug().Print(result.CopyResult)
		if result.ObjectOrNil != nil {
			ui.Debug().Print(sku.String(result.ObjectOrNil))
		}

		if !result.IsMissing() {
			return err
		}

		if result.ObjectOrNil.GetGenre() == genres.InventoryList {
			requestRetry = true
		}

		ui.Log().Print(
			"missing blob for list: %s",
			sku.String(result.ObjectOrNil),
		)

		// TODO switch to outputing object signatures
		clonedSku, _ := result.ObjectOrNil.CloneTransacted()
		listMissingSkus.Add(clonedSku)

		return err
	}

	importer := server.Repo.MakeImporter(
		importerOptions,
		sku.GetStoreOptionsRemoteTransfer(),
	)

	if err := server.Repo.ImportSeq(
		quiter.MakeSeqErrorFromSeq(list.All()),
		importer,
	); err != nil {
		if env_dir.IsErrBlobMissing(err) {
			requestRetry = true
		} else {
			response.Error(err)
			return response
		}
	}

	bufferedWriter, repoolBufferedWriter := pool.GetBufferedWriter(
		responseBuffer,
	)
	defer repoolBufferedWriter()

	inventoryListCoderCloset := local.GetInventoryListCoderCloset()

	if _, err := inventoryListCoderCloset.WriteTypedBlobToWriter(
		local,
		ids.GetOrPanic(local.GetImmutableConfigPublic().GetInventoryListTypeId()).TypeStruct,
		quiter.MakeSeqErrorFromSeq(listMissingSkus.All()),
		bufferedWriter,
	); err != nil {
		response.Error(err)
		return response
	}

	if err := bufferedWriter.Flush(); err != nil {
		response.Error(err)
		return response
	}

	if requestRetry {
		response.StatusCode = http.StatusExpectationFailed
	} else {
		response.StatusCode = http.StatusCreated
	}

	response.Body = ohio.NopCloser(responseBuffer)

	return response
}
