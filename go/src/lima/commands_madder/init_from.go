package commands_madder

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/src/foxtrot/fd"
	"code.linenisgreat.com/dodder/go/src/foxtrot/triple_hyphen_io"
	"code.linenisgreat.com/dodder/go/src/golf/blob_store_configs"
	"code.linenisgreat.com/dodder/go/src/india/env_local"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"code.linenisgreat.com/dodder/go/src/kilo/command_components_madder"
)

func init() {
	utility.AddCmd("init-from", &InitFrom{})
}

type InitFrom struct {
	command_components_madder.EnvBlobStore
	command_components_madder.Init
}

var _ interfaces.CommandComponentWriter = (*InitFrom)(nil)

func (cmd *InitFrom) SetFlagDefinitions(
	flagDefinitions interfaces.CLIFlagDefinitions,
) {
}

func (cmd InitFrom) Complete(
	req command.Request,
	envLocal env_local.Env,
	commandLine command.CommandLineInput,
) {
	// TODO support completion for config path
}

func (cmd *InitFrom) Run(req command.Request) {
	var blobStoreId blob_store_id.Id

	if err := blobStoreId.Set(req.PopArg("blob store name")); err != nil {
		errors.ContextCancelWithBadRequestError(req, err)
	}

	var configPathFD fd.FD

	if err := configPathFD.Set(req.PopArg("blob store config path")); err != nil {
		errors.ContextCancelWithBadRequestError(req, err)
	}

	req.AssertNoMoreArgs()

	envBlobStore := cmd.MakeEnvBlobStore(req)

	var typedConfig blob_store_configs.TypedConfig

	{
		var err error

		if typedConfig, err = triple_hyphen_io.DecodeFromFile(
			blob_store_configs.Coder,
			configPathFD.String(),
		); err != nil {
			envBlobStore.Cancel(err)
			return
		}
	}

	for {
		configUpgraded, ok := typedConfig.Blob.(blob_store_configs.ConfigUpgradeable)

		if !ok {
			break
		}

		typedConfig.Blob, typedConfig.Type = configUpgraded.Upgrade()
	}

	pathConfig := cmd.InitBlobStore(
		req,
		envBlobStore,
		blobStoreId,
		&typedConfig,
	)

	envBlobStore.GetUI().Printf("Wrote config to %s", pathConfig)
}
