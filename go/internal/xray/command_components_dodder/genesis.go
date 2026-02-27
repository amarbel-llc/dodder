package command_components_dodder

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/golf/env_ui"
	"code.linenisgreat.com/dodder/go/internal/hotel/env_dir"
	"code.linenisgreat.com/dodder/go/internal/india/env_local"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/internal/kilo/command_components_madder"
	"code.linenisgreat.com/dodder/go/internal/victor/local_working_copy"
)

type Genesis struct {
	env_repo.BigBang
	LocalWorkingCopy
	command_components_madder.Complete
}

var _ interfaces.CommandComponentWriter = (*Genesis)(nil)

func (cmd *Genesis) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.Var(
		&cmd.BigBang.InventoryListType,
		"inventory_list-type",
		"the type that will be used when creating inventory lists for this repo",
	)

	flagSet.BoolVar(
		&cmd.BigBang.OverrideXDGWithCwd,
		"override-xdg-with-cwd",
		false,
		"don't use XDG for this repo, and instead use the CWD and make a `.dodder` directory",
	)

	flagSet.StringVar(
		&cmd.BigBang.Yin,
		"yin",
		"",
		"File containing list of zettel id left parts",
	)

	flagSet.StringVar(
		&cmd.BigBang.Yang,
		"yang",
		"",
		"File containing list of zettel id right parts",
	)

	cmd.BigBang.SetDefaults()

	cmd.BigBang.GenesisConfig.Blob.SetFlagDefinitions(flagSet)

	cmd.BigBang.TypedBlobStoreConfig.Blob.SetFlagDefinitions(flagSet)

	flagSet.Var(
		&cmd.BigBang.PrivateKey,
		"private_key",
		"pre-existing private key markl.Id (use info-ssh_agent to list keys)",
	)

	flagSet.Var(
		cmd.Complete.GetFlagValueBlobIds(&cmd.BlobStoreId),
		"blob_store-id",
		"The name of the existing madder blob store to use",
	)
}

func (cmd Genesis) OnTheFirstDay(
	req command.Request,
	repoIdString string,
) *local_working_copy.Repo {
	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())
	envUI := env_ui.Make(
		req,
		config,
		config.Debug,
		env_ui.Options{},
	)

	var repoId ids.RepoId

	if err := repoId.Set(repoIdString); err != nil {
		envUI.Cancel(err)
	}

	cmd.GenesisConfig.Blob.SetRepoId(repoId)

	dir := env_dir.MakeDefaultAndInitialize(
		req,
		env_dir.XDGUtilityNameDodder,
		config.Debug,
		cmd.OverrideXDGWithCwd,
	)

	var envRepo env_repo.Env

	options := env_repo.Options{
		BasePath:                config.BasePath,
		PermitNoDodderDirectory: true,
	}

	{
		var err error

		if envRepo, err = env_repo.Make(
			env_local.Make(envUI, dir),
			options,
		); err != nil {
			envUI.Cancel(err)
		}
	}

	envRepo.Genesis(cmd.BigBang)

	return local_working_copy.Genesis(cmd.BigBang, envRepo)
}
