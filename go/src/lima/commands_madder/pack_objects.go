package commands_madder

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/src/hotel/tap_diagnostics"
	"code.linenisgreat.com/dodder/go/src/india/blob_stores"
	"code.linenisgreat.com/dodder/go/src/india/env_local"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"code.linenisgreat.com/dodder/go/src/kilo/command_components_madder"
	tap "github.com/amarbel-llc/tap-dancer/go"
)

func init() {
	utility.AddCmd("pack-objects", &PackObjects{})
}

type PackObjects struct {
	command_components_madder.EnvBlobStore
	command_components_madder.BlobStoreLocal

	DeleteLoose bool
	MaxPackSize uint64
}

var _ interfaces.CommandComponentWriter = (*PackObjects)(nil)

func (cmd PackObjects) Complete(
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

func (cmd *PackObjects) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.BoolVar(&cmd.DeleteLoose, "delete-loose", false,
		"validate archive then delete packed loose blobs")
	flagSet.Func("max-pack-size", "override max pack size in bytes (0 = unlimited)", func(v string) error {
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return err
		}
		cmd.MaxPackSize = n
		return nil
	})
}

func (cmd PackObjects) Run(req command.Request) {
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
		MaxPackSize:          cmd.MaxPackSize,
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

func (cmd PackObjects) doOne(
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
