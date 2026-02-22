package command_components_madder

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/delta/debug"
	"code.linenisgreat.com/dodder/go/src/echo/config_cli"
	"code.linenisgreat.com/dodder/go/src/foxtrot/repo_config_cli"
	"code.linenisgreat.com/dodder/go/src/golf/env_ui"
	"code.linenisgreat.com/dodder/go/src/hotel/env_dir"
	"code.linenisgreat.com/dodder/go/src/india/env_local"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"code.linenisgreat.com/dodder/go/src/juliett/env_repo"
)

type EnvBlobStore struct{}

func (cmd EnvBlobStore) MakeEnvBlobStore(
	req command.Request,
) env_repo.BlobStoreEnv {
	configAny := req.Utility.GetConfigAny()

	var debugOptions debug.Options
	var cliConfig domain_interfaces.CLIConfigProvider
	var envOptions env_ui.Options

	switch c := configAny.(type) {
	case *config_cli.Config:
		debugOptions = c.Debug
		cliConfig = c
		envOptions.CustomOut = c.CustomOut
		envOptions.CustomErr = c.CustomErr
	case *repo_config_cli.Config:
		debugOptions = c.Debug
		cliConfig = c
	default:
		panic(fmt.Sprintf("unsupported config type: %T", configAny))
	}

	dir := env_dir.MakeDefault(
		req,
		req.Utility.GetName(),
		debugOptions,
	)

	envUI := env_ui.Make(
		req,
		cliConfig,
		debugOptions,
		envOptions,
	)

	envLocal := env_local.Make(envUI, dir)

	return env_repo.MakeBlobStoreEnv(envLocal)
}
