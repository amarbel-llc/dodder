package commands_madder

import (
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"

	"code.linenisgreat.com/dodder/go/internal/charlie/tap_diagnostics"
	"code.linenisgreat.com/dodder/go/internal/delta/env_ui"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_local"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/hotel/command_components_madder"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	tap "github.com/amarbel-llc/bob/packages/tap-dancer/go"
)

func init() {
	utility.AddCmd("fsck", &Fsck{})
}

type Fsck struct {
	command_components_madder.EnvBlobStore
	command_components_madder.BlobStore
}

var _ command.CommandWithArgs = (*Fsck)(nil)

func (cmd *Fsck) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{{
			Name:        "store-ids",
			Description: "blob store IDs to verify (defaults to all configured stores)",
			Variadic:    true,
		}},
	}}
}

func (cmd Fsck) GetDescription() command.Description {
	return command.Description{
		Short: "verify blob store integrity",
		Long: "Verify the integrity of one or more blob stores by reading " +
			"every blob and recomputing its content-addressable digest. " +
			"Reports corrupt, missing, or unreadable blobs. With no " +
			"arguments, all configured stores are checked. Pass store IDs " +
			"to check specific stores. Output is TAP format with progress " +
			"updates every 3 seconds.",
	}
}

func (cmd Fsck) Complete(
	req command.Request,
	envLocal env_local.Env,
	commandLine command.CommandLineInput,
) {
	envBlobStore := cmd.MakeEnvBlobStore(req)

	for id, blobStore := range envBlobStore.GetBlobStores() {
		envLocal.GetOut().Printf("%s\t%s", id, blobStore.GetBlobStoreDescription())
	}
}

func (cmd Fsck) Run(req command.Request) {
	envBlobStore := cmd.MakeEnvBlobStore(req)

	blobStores := cmd.MakeBlobStoresFromIdsOrAll(req, envBlobStore)

	tw := tap.NewWriter(os.Stdout)

	for storeId, blobStore := range blobStores {
		tw.Comment(fmt.Sprintf("(blob_store: %s) starting fsck...", storeId))

		var count atomic.Uint32
		var errorCount atomic.Uint32
		var progressWriter env_ui.ProgressWriter

		if err := errors.RunChildContextWithPrintTicker(
			envBlobStore,
			func(ctx errors.Context) {
				for digest, err := range blobStore.AllBlobs() {
					errors.ContextContinueOrPanic(ctx)

					if err != nil {
						tw.NotOk("(unknown blob)", tap_diagnostics.FromError(err))
						errorCount.Add(1)
						count.Add(1)

						continue
					}

					count.Add(1)

					if !blobStore.HasBlob(digest) {
						tw.NotOk(fmt.Sprintf("%s", digest), map[string]string{"severity": "fail", "message": "blob missing"})
						errorCount.Add(1)

						continue
					}

					if err = blob_stores.VerifyBlob(
						ctx,
						blobStore,
						digest,
						io.MultiWriter(&progressWriter, io.Discard),
					); err != nil {
						tw.NotOk(fmt.Sprintf("%s", digest), tap_diagnostics.FromError(err))
						errorCount.Add(1)

						continue
					}

					tw.Ok(fmt.Sprintf("%s", digest))
				}
			},
			func(time time.Time) {
				tw.Comment(fmt.Sprintf(
					"(blob_store: %s) %d blobs / %s verified, %d errors",
					storeId,
					count.Load(),
					progressWriter.GetWrittenHumanString(),
					errorCount.Load(),
				))
			},
			3*time.Second,
		); err != nil {
			tw.BailOut(err.Error())
			envBlobStore.Cancel(err)
			return
		}

		tw.Comment(fmt.Sprintf(
			"(blob_store: %s) blobs verified: %d, bytes verified: %s",
			storeId,
			count.Load(),
			progressWriter.GetWrittenHumanString(),
		))
	}

	tw.Plan()
}
