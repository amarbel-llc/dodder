package commands_madder

import (
	"fmt"
	"io"
	"os"

	"code.linenisgreat.com/dodder/go/internal/_/blob_store_id"
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/charlie/tap_diagnostics"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_local"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/hotel/command_components_madder"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	tap "github.com/amarbel-llc/tap-dancer/go"
)

func init() {
	utility.AddCmd("pack-blobs", &PackBlobs{})
}

type PackBlobs struct {
	command_components_madder.EnvBlobStore
	command_components_madder.BlobStoreLocal

	DeleteLoose bool
	MaxPackSize ui.HumanReadableBytes
	Delta       bool
}

var _ interfaces.CommandComponentWriter = (*PackBlobs)(nil)

func (cmd PackBlobs) Complete(
	req command.Request,
	envLocal env_local.Env,
	commandLine command.CommandLineInput,
) {
	envBlobStore := cmd.MakeEnvBlobStore(req)
	blobStores := envBlobStore.GetBlobStores()

	for id, blobStore := range blobStores {
		envLocal.GetOut().Printf("%s\t%s", id, blobStore.GetBlobStoreDescription())
	}
}

func (cmd *PackBlobs) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.BoolVar(&cmd.DeleteLoose, "delete-loose", false,
		"validate archive then delete packed loose blobs")
	flagSet.BoolVar(&cmd.Delta, "delta", false,
		"enable delta compression during packing")
	flagSet.Var(&cmd.MaxPackSize, "max-pack-size",
		"override max pack size (e.g. 100M, 1G, 0 = unlimited)",
	)
}

func (cmd PackBlobs) Run(req command.Request) {
	envBlobStore := cmd.MakeEnvBlobStore(req)
	blobStore := envBlobStore.GetDefaultBlobStore()

	tw := tap.NewWriter(os.Stdout)

	var blobStoreId blob_store_id.Id
	storeIdString := ".default"
	blobFilter := make(map[string]domain_interfaces.MarklId)

	sawStdin := false

	for _, arg := range req.PopArgs() {
		switch {
		case arg == "-" && sawStdin:
			tw.Comment("'-' passed in more than once. Ignoring")
			continue

		case arg == "-":
			sawStdin = true
		}

		resolved := command_components_madder.ResolveFileOrBlobStoreId(arg)

		if resolved.Err != nil {
			tw.NotOk(arg, tap_diagnostics.FromError(resolved.Err))
			continue
		}

		if resolved.IsStoreSwitch {
			blobStoreId = resolved.BlobStoreId
			blobStore = envBlobStore.GetBlobStore(blobStoreId)
			storeIdString = blobStoreId.String()
			tw.Comment(fmt.Sprintf("switched to blob store: %s", storeIdString))
			continue
		}

		blobId, err := cmd.doOne(blobStore, resolved.BlobReader)
		if err != nil {
			tw.NotOk(arg, tap_diagnostics.FromError(err))
			continue
		}

		if blobId.IsNull() {
			tw.Skip(arg, "null digest")
			continue
		}

		tw.Ok(fmt.Sprintf("%s %s", blobId, arg))
		blobFilter[blobId.String()] = blobId
	}

	if len(blobFilter) == 0 {
		tw.Plan()
		return
	}

	packable, ok := blobStore.BlobStore.(blob_stores.PackableArchive)
	if !ok {
		tw.NotOk(
			fmt.Sprintf("pack %s", storeIdString),
			map[string]string{
				"severity": "fail",
				"message":  "not packable",
			},
		)
		tw.Plan()
		return
	}

	if err := packable.Pack(blob_stores.PackOptions{
		Context:              req,
		DeleteLoose:          cmd.DeleteLoose,
		DeletionPrecondition: blob_stores.NopDeletionPrecondition(),
		BlobFilter:           blobFilter,
		MaxPackSize:          cmd.MaxPackSize.GetByteCount(),
		Delta:                cmd.Delta,
		TapWriter:            tw,
	}); err != nil {
		tw.NotOk(
			fmt.Sprintf("pack %s", storeIdString),
			tap_diagnostics.FromError(err),
		)
		tw.Plan()
		return
	}

	tw.Ok(fmt.Sprintf("pack %s", storeIdString))
	tw.Plan()
}

func (cmd PackBlobs) doOne(
	blobStore blob_stores.BlobStoreInitialized,
	blobReader domain_interfaces.BlobReader,
) (blobId domain_interfaces.MarklId, err error) {
	defer errors.DeferredCloser(&err, blobReader)

	var writeCloser domain_interfaces.BlobWriter

	if writeCloser, err = blobStore.MakeBlobWriter(nil); err != nil {
		err = errors.Wrap(err)
		return blobId, err
	}

	defer errors.DeferredCloser(&err, writeCloser)

	if _, err = io.Copy(writeCloser, blobReader); err != nil {
		err = errors.Wrap(err)
		return blobId, err
	}

	blobId = writeCloser.GetMarklId()

	return blobId, err
}
