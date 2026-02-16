package remote_http

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/src/bravo/markl_io"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/echo/directory_layout"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/golf/blob_store_configs"
	"code.linenisgreat.com/dodder/go/src/hotel/env_dir"
	"code.linenisgreat.com/dodder/go/src/india/blob_stores"
)

func (client *client) GetBlobStore() blob_stores.BlobStoreInitialized {
	return blob_stores.BlobStoreInitialized{
		ConfigNamed: blob_store_configs.ConfigNamed{
			Path: directory_layout.MakeBlobStorePath(
				blob_store_id.MakeWithLocation(
					"remote",
					blob_store_id.LocationTypeUnknown,
				),
				"",
				"remote",
			),
		},
		BlobStore: client,
	}
}

func (client *client) GetBlobStoreConfig() blob_store_configs.Config {
	panic(errors.Err501NotImplemented)
}

func (client *client) GetDefaultHashType() domain_interfaces.FormatHash {
	panic(errors.Err501NotImplemented)
}

func (client *client) HasBlob(blobId domain_interfaces.MarklId) (ok bool) {
	var request *http.Request

	{
		var err error

		if request, err = client.newRequest(
			"HEAD",
			fmt.Sprintf("/blobs/%s", blobId),
			nil,
		); err != nil {
			client.GetEnv().Cancel(err)
		}
	}

	var response *http.Response

	{
		var err error

		if response, err = client.http.Do(request); err != nil {
			client.GetEnv().Cancel(err)
		}
	}

	ok = response.StatusCode == http.StatusNoContent

	return ok
}

func (client *client) MakeBlobReader(
	blobId domain_interfaces.MarklId,
) (reader domain_interfaces.BlobReader, err error) {
	var request *http.Request

	if request, err = client.newRequest(
		"GET",
		fmt.Sprintf("/blobs/%s", blobId),
		nil,
	); err != nil {
		err = errors.Wrap(err)
		return reader, err
	}

	var response *http.Response

	if response, err = client.http.Do(request); err != nil {
		err = errors.Wrap(err)
		return reader, err
	}

	switch {
	case response.StatusCode == http.StatusNotFound:
		err = env_dir.ErrBlobMissing{
			BlobId: blobId,
		}

	case response.StatusCode >= 300:
		err = ReadErrorFromBody(response)

	default:
		var hashType markl.FormatHash

		if hashType, err = markl.GetFormatHashOrError(
			blobId.GetMarklFormat().GetMarklFormatId(),
		); err != nil {
			err = errors.Wrap(err)
			return reader, err
		}

		hash, _ := hashType.Get()
		reader = markl_io.MakeReadCloser(
			hash,
			response.Body,
		)
	}

	return reader, err
}

func (client *client) WriteBlobToRemote(
	localBlobStore domain_interfaces.BlobStore,
	expected domain_interfaces.MarklId,
) (err error) {
	// Closed by the http client's transport (our roundtripper calling
	// request.Write)
	var reader domain_interfaces.BlobReader

	if reader, err = localBlobStore.MakeBlobReader(
		expected,
	); err != nil {
		if env_dir.IsErrBlobMissing(err) {
			// TODO make an option to collect this error at the present it, and
			// an
			// option to fetch it from another remote store
			ui.Err().Printf("Blob missing locally: %q", expected)
			err = nil
		} else {
			err = errors.Wrap(err)
		}

		return err
	}

	var request *http.Request

	if request, err = client.newRequest(
		"POST",
		"/blobs",
		reader,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	request.TransferEncoding = []string{"chunked"}

	var response *http.Response

	if response, err = client.http.Do(request); err != nil {
		err = errors.ErrorWithStackf("failed to read response: %w", err)
		return err
	}

	if err = ReadErrorFromBodyOnNot(response, http.StatusCreated); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var digestString strings.Builder

	if _, err = io.Copy(&digestString, response.Body); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = response.Body.Close(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var actual markl.Id

	if err = actual.Set(
		strings.TrimSpace(digestString.String()),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = markl.AssertEqual(expected, &actual); err != nil {
		ui.Debug().Print(err)
		err = errors.Wrapf(err, "Raw Blob Id: %q", digestString.String())
		return err
	}

	return err
}

//   _   _       _
//  | \ | | ___ | |_
//  |  \| |/ _ \| __|
//  | |\  | (_) | |_
//  |_| \_|\___/ \__|
//
//   ___                 _                           _           _
//  |_ _|_ __ ___  _ __ | | ___ _ __ ___   ___ _ __ | |_ ___  __| |
//   | || '_ ` _ \| '_ \| |/ _ \ '_ ` _ \ / _ \ '_ \| __/ _ \/ _` |
//   | || | | | | | |_) | |  __/ | | | | |  __/ | | | ||  __/ (_| |
//  |___|_| |_| |_| .__/|_|\___|_| |_| |_|\___|_| |_|\__\___|\__,_|
//                |_|

func (client *client) GetBlobStoreDescription() string {
	panic(errors.Err501NotImplemented)
}

func (client *client) GetBlobIOWrapper() domain_interfaces.BlobIOWrapper {
	panic(errors.Err501NotImplemented)
}

func (client *client) AllBlobs() interfaces.SeqError[domain_interfaces.MarklId] {
	panic(errors.Err501NotImplemented)
}

func (client *client) MakeBlobWriter(
	marklHashType domain_interfaces.FormatHash,
) (domain_interfaces.BlobWriter, error) {
	panic(errors.Err501NotImplemented)
}
