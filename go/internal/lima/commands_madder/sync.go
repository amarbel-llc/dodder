package commands_madder

import (
	"fmt"
	"os"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/ui"
	"code.linenisgreat.com/dodder/go/internal/hotel/env_dir"
	"code.linenisgreat.com/dodder/go/internal/hotel/tap_diagnostics"
	"code.linenisgreat.com/dodder/go/internal/india/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/blob_transfers"
	"code.linenisgreat.com/dodder/go/internal/kilo/command_components_madder"
	tap "github.com/amarbel-llc/tap-dancer/go"
)

func init() {
	utility.AddCmd("sync", &Sync{})
}

type Sync struct {
	command_components_madder.EnvBlobStore
	command_components_madder.BlobStore

	AllowRehashing bool
	Limit          int
}

var _ interfaces.CommandComponentWriter = (*Sync)(nil)

func (cmd *Sync) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.BoolVar(
		&cmd.AllowRehashing,
		"allow-rehashing",
		false,
		"allow syncing to stores with a different hash type (source digests not preserved in single-hash destinations)",
	)

	flagSet.IntVar(
		&cmd.Limit,
		"limit",
		0,
		"number of blobs to sync before stopping. 0 means don't stop (full consent)",
	)
}

// TODO add completion for blob store id's

func (cmd Sync) Run(req command.Request) {
	envBlobStore := cmd.MakeEnvBlobStore(req)

	source, destinations := cmd.MakeSourceAndDestinationBlobStoresFromIdsOrAll(
		req,
		envBlobStore,
	)

	cmd.runStore(req, envBlobStore, source, destinations)
}

func (cmd Sync) runStore(
	req command.Request,
	envBlobStore env_repo.BlobStoreEnv,
	source blob_stores.BlobStoreInitialized,
	destination blob_stores.BlobStoreMap,
) {
	tw := tap.NewWriter(os.Stdout)

	if len(destination) == 0 {
		tw.BailOut("only one blob store, nothing to sync")

		errors.ContextCancelWithBadRequestf(
			req,
			"only one blob store, nothing to sync",
		)

		return
	}

	sourceHashType := source.GetDefaultHashType()
	useDestinationHashType := false

	for _, dst := range destination {
		dstHashType := dst.GetDefaultHashType()

		if sourceHashType.GetMarklFormatId() == dstHashType.GetMarklFormatId() {
			continue
		}

		_, isAdder := dst.GetBlobStore().(domain_interfaces.BlobForeignDigestAdder)

		if !isAdder && !cmd.AllowRehashing {
			if !envBlobStore.Confirm(
				fmt.Sprintf(
					"Destination %q uses %s but source uses %s. Rehashing will not preserve source digests. Continue?",
					dst.GetId(),
					dstHashType.GetMarklFormatId(),
					sourceHashType.GetMarklFormatId(),
				),
				"",
			) {
				errors.ContextCancelWithBadRequestf(
					req,
					"cross-hash sync refused: destination %q uses %s, source uses %s. Use -allow-rehashing to skip this check",
					dst.GetId(),
					dstHashType.GetMarklFormatId(),
					sourceHashType.GetMarklFormatId(),
				)

				return
			}
		}

		useDestinationHashType = true
	}

	blobImporter := blob_transfers.MakeBlobImporter(
		envBlobStore,
		source,
		destination,
	)

	blobImporter.UseDestinationHashType = useDestinationHashType

	var lastBytesWritten int64

	blobImporter.CopierDelegate = func(result sku.BlobCopyResult) error {
		bytesWritten, _ := result.GetBytesWrittenAndState()
		lastBytesWritten = bytesWritten
		return nil
	}

	defer req.Must(
		func(_ interfaces.ActiveContext) error {
			tw.Comment(fmt.Sprintf(
				"Successes: %d, Failures: %d, Ignored: %d, Total: %d",
				blobImporter.Counts.Succeeded,
				blobImporter.Counts.Failed,
				blobImporter.Counts.Ignored,
				blobImporter.Counts.Total,
			))

			tw.Plan()

			return nil
		},
	)

	for blobId, errIter := range source.AllBlobs() {
		lastBytesWritten = 0

		if errIter != nil {
			tw.NotOk(
				fmt.Sprintf("%s", blobId),
				tap_diagnostics.FromError(errIter),
			)

			continue
		}

		if err := blobImporter.ImportBlobIfNecessary(blobId, nil); err != nil {
			if env_dir.IsErrBlobAlreadyExists(err) {
				tw.Ok(formatBlobTestPoint(blobId, lastBytesWritten))
			} else {
				tw.NotOk(
					formatBlobTestPoint(blobId, lastBytesWritten),
					tap_diagnostics.FromError(err),
				)
			}
		} else {
			tw.Ok(formatBlobTestPoint(blobId, lastBytesWritten))
		}

		if cmd.Limit > 0 &&
			(blobImporter.Counts.Succeeded+blobImporter.Counts.Failed) > cmd.Limit {
			tw.Comment("limit hit, stopping")
			break
		}
	}
}

func formatBlobTestPoint(
	blobId domain_interfaces.MarklId,
	bytesWritten int64,
) string {
	if bytesWritten > 0 {
		return fmt.Sprintf("%s (%s)", blobId, ui.GetHumanBytesStringOrError(bytesWritten))
	}

	return fmt.Sprintf("%s", blobId)
}
