package commands_madder

import (
	"encoding/json"
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/internal/india/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/kilo/command_components_madder"
)

func init() {
	utility.AddCmd("read", &Read{})
}

type Read struct {
	command_components_madder.EnvBlobStore
}

type readBlobEntry struct {
	Blob  string `json:"blob"`
	Store string `json:"store,omitempty"`
}

func (cmd Read) Run(dep command.Request) {
	envBlobStore := cmd.MakeEnvBlobStore(dep)

	decoder := json.NewDecoder(envBlobStore.GetInFile())
	blobStore := envBlobStore.GetDefaultBlobStore()

	for {
		var entry readBlobEntry

		if err := decoder.Decode(&entry); err != nil {
			if errors.IsEOF(err) {
				err = nil
			} else {
				envBlobStore.Cancel(err)
			}

			return
		}

		if entry.Store != "" {
			var storeId blob_store_id.Id

			if err := storeId.Set(entry.Store); err != nil {
				envBlobStore.Cancel(err)
				return
			}

			blobStore = envBlobStore.GetBlobStore(storeId)
		}

		{
			var err error

			if _, err = cmd.readOneBlob(blobStore, entry); err != nil {
				envBlobStore.Cancel(err)
			}
		}
	}
}

func (Read) readOneBlob(
	blobStore blob_stores.BlobStoreInitialized,
	entry readBlobEntry,
) (digest domain_interfaces.MarklId, err error) {
	var writeCloser domain_interfaces.BlobWriter

	if writeCloser, err = blobStore.MakeBlobWriter(
		nil,
	); err != nil {
		err = errors.Wrap(err)
		return digest, err
	}

	defer errors.DeferredCloser(&err, writeCloser)

	if _, err = io.Copy(writeCloser, strings.NewReader(entry.Blob)); err != nil {
		err = errors.Wrap(err)
		return digest, err
	}

	digest = writeCloser.GetMarklId()

	return digest, err
}
