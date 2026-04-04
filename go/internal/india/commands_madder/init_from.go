package commands_madder

import (
	"fmt"
	"os"

	"code.linenisgreat.com/dodder/go/internal/alfa/blob_store_id"
	"code.linenisgreat.com/dodder/go/internal/charlie/fd"
	"code.linenisgreat.com/dodder/go/internal/charlie/hyphence"
	"code.linenisgreat.com/dodder/go/internal/delta/blob_store_configs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_local"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/hotel/command_components_madder"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	tap "github.com/amarbel-llc/bob/packages/tap-dancer/go"
)

func init() {
	utility.AddCmd("init-from", &InitFrom{})
}

type InitFrom struct {
	command_components_madder.EnvBlobStore
	command_components_madder.Init
}

var (
	_ interfaces.CommandComponentWriter = (*InitFrom)(nil)
	_ command.CommandWithArgs           = (*InitFrom)(nil)
)

func (cmd *InitFrom) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{
			{
				Name:        "store-name",
				Description: "name for the new blob store",
				Required:    true,
			},
			{
				Name:        "config-path",
				Description: "path to the blob store configuration file",
				Required:    true,
			},
		},
	}}
}

func (cmd InitFrom) GetDescription() command.Description {
	return command.Description{
		Short: "initialize a blob store from a configuration",
	}
}

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

	tw := tap.NewWriter(os.Stdout)

	envBlobStore := cmd.MakeEnvBlobStore(req)

	var typedConfig blob_store_configs.TypedConfig

	{
		var err error

		if typedConfig, err = hyphence.DecodeFromFile(
			blob_store_configs.Coder,
			configPathFD.String(),
		); err != nil {
			tw.NotOk(
				fmt.Sprintf("init-from %s", configPathFD.String()),
				map[string]string{
					"severity": "fail",
					"message":  err.Error(),
				},
			)
			tw.Plan()
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

	tw.Ok(fmt.Sprintf("init-from %s", pathConfig.GetConfig()))
	tw.Plan()
}
