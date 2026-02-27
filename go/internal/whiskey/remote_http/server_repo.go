package remote_http

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/ohio"
	"code.linenisgreat.com/dodder/go/internal/bravo/ui"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
	"code.linenisgreat.com/dodder/go/internal/charlie/tridex"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/log_remote_inventory_lists"
)

func (server *Server) writeInventoryList(
	request Request,
	listObject *sku.Transacted,
) (response Response) {
	logRemoteInventoryLists := log_remote_inventory_lists.Make(
		request.ctx,
		server.Repo.GetEnvRepo(),
	)

	if listObject == nil {
		panic("nil list object")
	}

	if listObject.GetGenre() != genres.InventoryList {
		response.Error(genres.MakeErrUnsupportedGenre(listObject.GetGenre()))
		return response
	}

	blobStore := server.Repo.GetBlobStore()

	if blobStore.HasBlob(listObject.GetBlobDigest()) {
		response.StatusCode = http.StatusFound
		return response
	}

	expected := listObject.GetBlobDigest()

	pubBase64 := request.request.Header.Get(headerRepoPublicKey)

	var logEntry log_remote_inventory_lists.Entry

	if pubBase64 != "" {
		{
			var err error

			var bites []byte

			if bites, err = base64.URLEncoding.DecodeString(pubBase64); err != nil {
				response.Error(err)
				return response
			}

			var pubkey markl.Id

			if err = pubkey.SetMarklId(
				markl.FormatIdEd25519Pub,
				bites,
			); err != nil {
				response.Error(err)
				return response
			}

			logEntry.PublicKey = pubkey
		}

		logEntry.EntryType = log_remote_inventory_lists.EntryTypeReceived
		logEntry.Transacted = listObject

		var sig markl.Id

		if err := sig.Set(request.request.Header.Get(headerRepoSig)); err != nil {
			response.Error(err)
			return response
		}

		if err := logEntry.PublicKey.Verify(
			expected,
			sig,
		); err != nil {
			response.Error(err)
			return response
		}
	}

	if len(logEntry.PublicKey.GetBytes()) > 0 {
		if err := logRemoteInventoryLists.Exists(
			logEntry,
		); errors.IsErrNotFound(err) && err != nil {
			err = nil
		} else if err != nil {
			response.Error(err)
			return response
		} else {
			response.StatusCode = http.StatusFound
			return response
		}
	}

	typedInventoryListStore := server.Repo.GetInventoryListCoderCloset()

	var blobWriter domain_interfaces.BlobWriter

	{
		var err error

		if blobWriter, err = blobStore.MakeBlobWriter(nil); err != nil {
			response.Error(err)
			return response
		}
	}

	seqInventoryListSkus := typedInventoryListStore.IterInventoryListBlobSkusFromReader(
		listObject.GetType().ToType(),
		bufio.NewReader(io.TeeReader(request.Body, blobWriter)),
	)

	b := bytes.NewBuffer(nil)
	writtenNeededBlobs := tridex.Make()

	{
		count := 0

		for sk, err := range seqInventoryListSkus {
			errors.ContextContinueOrPanic(server.Repo.GetEnv())

			if err != nil {
				response.Error(err)
				return response
			}

			blobDigest := sk.GetBlobDigest()

			var ok bool
			ok, err = server.blobCache.HasBlob(blobDigest)
			if err != nil {
				response.Error(err)
				return response
			}

			blobDigestString := blobDigest.String()

			if ok || writtenNeededBlobs.ContainsExpansion(blobDigestString) {
				continue
			}

			ui.Log().Printf("missing blob: %s", blobDigest)

			fmt.Fprintf(b, "%s\n", blobDigest)
			writtenNeededBlobs.Add(blobDigestString)
			count++
		}

		ui.Err().Printf("missing blobs: %d", count)
	}

	if err := blobWriter.Close(); err != nil {
		response.Error(err)
		return response
	}

	actual := blobWriter.GetMarklId()

	if err := markl.AssertEqual(expected, actual); err != nil {
		ui.Err().Printf(
			"received list has different sha: expected: %s, actual: %s",
			expected,
			actual,
		)

		// response.ErrorWithStatus(http.StatusBadRequest, err)
		// return
	}

	ui.Log().Printf("list sha matches: %s", expected)

	// TODO make merge conflicts impossible

	response.StatusCode = http.StatusCreated
	response.Body = ohio.NopCloser(b)

	if err := server.Repo.GetObjectStore().Commit(
		listObject,
		sku.CommitOptions{},
	); err != nil {
		response.Error(err)
		return response
	}

	if len(logEntry.PublicKey.GetBytes()) > 0 {
		if err := logRemoteInventoryLists.Append(
			logEntry,
		); err != nil {
			response.Error(err)
			return response
		}
	}

	return response
}
