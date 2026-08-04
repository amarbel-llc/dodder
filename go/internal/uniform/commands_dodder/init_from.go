package commands_dodder

import (
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/repo_id"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/madder/go/pkgs/env_local"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

func init() {
	utility.AddCmd(
		"init-from",
		&InitFrom{
			Genesis: command_components_dodder.Genesis{
				BigBang: env_repo.BigBang{
					ExcludeDefaultType: true,
				},
			},
		},
	)
}

// InitFrom is the repo half of the RFC-0007 copy-migration pattern
// (madder's `init-from --from-store` proved it store-side): create a NEW
// repo with the current config flavor — which mints a fresh uuidv7
// instance identity — while KEEPING the source repo's keypair (the uuid
// is the identity, the pubkey its attestor; same logical repo, so same
// keys), then populate it with the source's full object history and seed
// its config log from the source's current config. The source is opened
// read-only and never modified (the dodder#363 migrate-repo-layout copy
// precedent); the user deletes it when satisfied.
type InitFrom struct {
	command_components_dodder.Genesis
	command_components_dodder.Query

	From string
}

var (
	_ interfaces.CommandComponentWriter = (*InitFrom)(nil)
	_ command.CommandWithArgs           = (*InitFrom)(nil)
)

func (cmd *InitFrom) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{
		{Args: []command.Arg{{
			Name:        "repo-id",
			Description: "location handle for the new local repository (scope via spelling: name=user, .name=cwd, //name=system)",
			Required:    true,
		}}},
		cmd.Query.GetArgGroup(),
	}
}

func (cmd InitFrom) GetDescription() command.Description {
	return command.Description{
		Short: "create a new repo copy-migrated from an existing local repo (fresh instance identity, same keys; source untouched)",
	}
}

func (cmd *InitFrom) SetFlagDefinitions(
	flagDefinitions interfaces.CLIFlagDefinitions,
) {
	cmd.Genesis.SetFlagDefinitions(flagDefinitions)
	cmd.Query.SetFlagDefinitions(flagDefinitions)

	flagDefinitions.StringVar(
		&cmd.From,
		"from",
		"",
		"path to the source repo's root directory (the tree holding its repos/default nest); the source is never modified",
	)
}

func (cmd *InitFrom) Run(req command.Request) {
	if cmd.From == "" {
		req.Cancel(errors.BadRequestf("-from is required"))
		return
	}

	absFrom, err := filepath.Abs(cmd.From)
	if err != nil {
		req.Cancel(errors.Wrap(err))
		return
	}

	source := cmd.openSource(req, absFrom)

	// Harvest the source's genesis identity BEFORE genesis: the migrated
	// repo keeps the source's keypair, in-graph repo id, and type
	// choices. What it does NOT keep is the instance identity — BigBang's
	// GenesisConfig came from genesis_configs.Default(), which minted a
	// fresh uuidv7; that fresh mint IS the migration's identity cut.
	sourceGenesis := source.GetEnvRepo().GetConfigPrivate().Blob

	cmd.BigBang.PrivateKey.ResetWithMarklId(sourceGenesis.GetPrivateKey())

	genesisBlob := cmd.BigBang.GenesisConfig.Blob
	genesisBlob.SetRepoId(sourceGenesis.GetRepoId())
	genesisBlob.SetInventoryListTypeId(sourceGenesis.GetInventoryListTypeId())
	genesisBlob.SetObjectSigMarklTypeId(sourceGenesis.GetObjectSigMarklTypeId())

	cmd.SetLocationFromPositionalRequired(req, "new repo id")

	local := cmd.OnTheFirstDay(req)

	queryGroup := cmd.MakeQueryIncludingWorkspace(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionDefaultSigil(
				ids.SigilHistory,
				ids.SigilHidden,
			),
			queries.BuilderOptionDefaultGenres(genres.InventoryList),
		),
		local,
		req.PopArgs(),
	)

	if err := local.PullQueryGroupFromRemote(
		source,
		queryGroup,
		repo.ImporterOptions{}.WithPrintCopies(true),
	); err != nil {
		req.Cancel(err)
		return
	}

	// Config is repo-local (FDR 0020): the pull never transfers it, so
	// seed the migrated repo's config log from the source's current
	// config, same as clone's direct transfer.
	seedConfigLogFromLocalSource(req, local, source)
}

// openSource opens the source repo read-only at its root path, standalone
// (before any genesis) — the same construction Remote.MakeRemoteFromBlob
// uses for a BlobOverridePath direct remote. The source repo within the
// root must use the default name (`repos/default`), which is what
// migrate-repo-layout produces for legacy trees.
func (cmd *InitFrom) openSource(
	req command.Request,
	absFrom string,
) *local_working_copy.Repo {
	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())

	ownDir := env_dir.MakeWithXDGRootOverrideHomeAndInitialize(
		req,
		absFrom,
		req.Utility.GetName(),
		config.Debug,
		repo_id.DefaultName,
	)

	madderDir := env_dir.MakeWithXDGRootOverrideHomeAndInitialize(
		req,
		absFrom,
		command_components_dodder.XDGUtilityNameMadder,
		config.Debug,
		repo_id.DefaultName,
	)

	envUI := env_ui.Make(
		req,
		config,
		config.Debug,
		env_ui.Options{UIPrintingPrefix: "source"},
	)

	return local_working_copy.Make(
		env_local.Make(envUI, ownDir),
		env_local.Make(envUI, madderDir),
		local_working_copy.OptionsEmpty,
	)
}
