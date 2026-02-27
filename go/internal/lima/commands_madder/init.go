package commands_madder

import (
	"fmt"
	"os"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/lib/charlie/compression_type"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/blob_store_configs"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/kilo/command_components_madder"
	tap "github.com/amarbel-llc/tap-dancer/go"
)

func init() {
	utility.AddCmd("init", &Init{
		tipe: ids.GetOrPanic(ids.TypeTomlBlobStoreConfigVCurrent).TypeStruct,
		blobStoreConfig: &blob_store_configs.DefaultType{
			CompressionType:   compression_type.CompressionTypeDefault,
			LockInternalFiles: true,
		},
	})

	utility.AddCmd("init-pointer", &Init{
		tipe: ids.GetOrPanic(
			ids.TypeTomlBlobStoreConfigPointerV0,
		).TypeStruct,
		blobStoreConfig: &blob_store_configs.TomlPointerV0{},
	})

	utility.AddCmd("init-sftp-explicit", &Init{
		tipe: ids.GetOrPanic(
			ids.TypeTomlBlobStoreConfigSftpExplicitV0,
		).TypeStruct,
		blobStoreConfig: &blob_store_configs.TomlSFTPV0{},
	})

	utility.AddCmd("init-sftp-ssh_config", &Init{
		tipe: ids.GetOrPanic(
			ids.TypeTomlBlobStoreConfigSftpViaSSHConfigV0,
		).TypeStruct,
		blobStoreConfig: &blob_store_configs.TomlSFTPViaSSHConfigV0{},
	})

	utility.AddCmd("init-inventory-archive", &Init{
		tipe: ids.GetOrPanic(
			ids.TypeTomlBlobStoreConfigInventoryArchiveVCurrent,
		).TypeStruct,
		blobStoreConfig: &blob_store_configs.TomlInventoryArchiveV2{
			Delta: blob_store_configs.DeltaConfig{
				Enabled:     false,
				Algorithm:   "bsdiff",
				MinBlobSize: 256,
				MaxBlobSize: 10485760,
				SizeRatio:   2.0,
			},
		},
	})

	utility.AddCmd("init-inventory-archive-v1", &Init{
		tipe: ids.GetOrPanic(
			ids.TypeTomlBlobStoreConfigInventoryArchiveV1,
		).TypeStruct,
		blobStoreConfig: &blob_store_configs.TomlInventoryArchiveV1{
			Delta: blob_store_configs.DeltaConfig{
				Enabled:     false,
				Algorithm:   "bsdiff",
				MinBlobSize: 256,
				MaxBlobSize: 10485760,
				SizeRatio:   2.0,
			},
		},
	})

	utility.AddCmd("init-inventory-archive-v0", &Init{
		tipe: ids.GetOrPanic(
			ids.TypeTomlBlobStoreConfigInventoryArchiveV0,
		).TypeStruct,
		blobStoreConfig: &blob_store_configs.TomlInventoryArchiveV0{},
	})
}

type Init struct {
	tipe            ids.TypeStruct
	blobStoreConfig blob_store_configs.ConfigMutable

	command_components_madder.EnvBlobStore
	command_components_madder.Init
}

var _ interfaces.CommandComponentWriter = (*Init)(nil)

func (cmd *Init) SetFlagDefinitions(
	flagDefinitions interfaces.CLIFlagDefinitions,
) {
	cmd.blobStoreConfig.SetFlagDefinitions(flagDefinitions)
}

func (cmd *Init) Run(req command.Request) {
	var blobStoreId blob_store_id.Id

	if err := blobStoreId.Set(req.PopArg("blob store id")); err != nil {
		errors.ContextCancelWithBadRequestError(req, err)
	}

	req.AssertNoMoreArgs()

	tw := tap.NewWriter(os.Stdout)

	envBlobStore := cmd.MakeEnvBlobStore(req)

	pathConfig := cmd.InitBlobStore(
		req,
		envBlobStore,
		blobStoreId,
		&blob_store_configs.TypedConfig{
			Type: cmd.tipe,
			Blob: cmd.blobStoreConfig,
		},
	)

	tw.Ok(fmt.Sprintf("init %s", pathConfig.GetConfig()))
	tw.Plan()
}
