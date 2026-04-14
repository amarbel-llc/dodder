package commands_madder

import (
	"encoding/json"
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/hotel/command_components_madder"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"github.com/amarbel-llc/madder/go/pkgs/blob_store_id"
)

func init() {
	utility.AddCmd("read", &Read{})
}

type Read struct {
	command_components_madder.EnvBlobStore
}

var _ command.CommandWithArgs = (*Read)(nil)

func (cmd *Read) GetArgs() []command.ArgGroup { return nil }

func (cmd Read) GetDescription() command.Description {
	return command.Description{
		Short: "read blobs from JSON on stdin",
		Long: "Read JSON objects from stdin and write each blob value into " +
			"the content-addressable store. Each JSON object must have a " +
			"\"blob\" field containing the content to store. An optional " +
			"\"store\" field switches the target blob store for that entry. " +
			"This command is the programmatic counterpart to 'write', " +
			"accepting structured input rather than file paths.",
	}
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
