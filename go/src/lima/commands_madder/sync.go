package commands_madder

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/hotel/env_dir"
	"code.linenisgreat.com/dodder/go/src/india/blob_stores"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"code.linenisgreat.com/dodder/go/src/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/kilo/blob_transfers"
	"code.linenisgreat.com/dodder/go/src/kilo/command_components_madder"
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
	// TODO output TAP
	ui.Out().Print("Blob Stores:")

	if len(destination) == 0 {
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

	blobImporter.CopierDelegate = sku.MakeBlobCopierDelegate(
		envBlobStore.GetUI(),
		false,
	)

	defer req.Must(
		func(_ interfaces.ActiveContext) error {
			ui.Err().Printf(
				"Successes: %d, Failures: %d, Ignored: %d, Total: %d",
				blobImporter.Counts.Succeeded,
				blobImporter.Counts.Failed,
				blobImporter.Counts.Ignored,
				blobImporter.Counts.Total,
			)

			return nil
		},
	)

	for blobId, errIter := range source.AllBlobs() {
		if errIter != nil {
			ui.Err().Print(errIter)
			continue
		}

		if err := blobImporter.ImportBlobIfNecessary(blobId, nil); err != nil {
			var errNotEqual markl.ErrNotEqual

			if errors.As(err, &errNotEqual) {
				ui.Err().Printf(
					"%q -> %q",
					errNotEqual.Expected,
					errNotEqual.Actual,
				)
			} else if !env_dir.IsErrBlobAlreadyExists(err) {
				ui.Err().Print(err)
			}
		}

		if cmd.Limit > 0 &&
			(blobImporter.Counts.Succeeded+blobImporter.Counts.Failed) > cmd.Limit {
			ui.Err().Print("limit hit, stopping")
			break
		}
	}
}
