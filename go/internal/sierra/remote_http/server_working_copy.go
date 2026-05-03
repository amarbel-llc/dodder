package remote_http

import (
	"bytes"
	"net/http"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"github.com/amarbel-llc/madder/go/pkgs/markl_io"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/0/pool"
	"code.linenisgreat.com/dodder/go/lib/alfa/ohio"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

func (server *Server) writeInventoryListTypedBlobLocalWorkingCopy(
	local *local_working_copy.Repo,
	request Request,
) (response Response) {
	listCoderCloset := server.Repo.GetInventoryListCoderCloset()

	blobStore := server.Repo.GetBlobStore()

	hashFormat := blobStore.GetDefaultHashType()
	hash, repoolHash := hashFormat.GetHash()
	defer repoolHash()

	digestWriter := markl_io.MakeWriter(hash, nil)

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

		clonedObj, _ := result.ObjectOrNil.CloneTransacted() //repool:owned
		listMissingObjects.Add(clonedObj)

		return err
	}

	importer := server.Repo.MakeImporter(
		importerOptions,
		sku.GetStoreOptionsRemoteTransfer(),
	)

	var claimedBlobDigest markl.Id

	seq := listCoderCloset.AllDecodedObjectsFromStreamWithBlobDigestValidation(
		request.Body,
		nil,
		digestWriter,
		&claimedBlobDigest,
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

	computedBlobDigest := digestWriter.GetMarklId()

	ui.Log().Printf(
		"received inventory list blob: %s",
		computedBlobDigest,
	)

	if claimedBlobDigest.IsNull() {
		response.Error(errors.Errorf(
			"inventory list missing required blob digest in metadata",
		))
		return response
	}

	if err := markl.AssertEqual(&claimedBlobDigest, computedBlobDigest); err != nil {
		response.Error(errors.Wrapf(
			err,
			"inventory list blob digest mismatch",
		))
		return response
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
