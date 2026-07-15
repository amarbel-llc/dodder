package command_components_dodder

import (
	"bufio"

	"code.linenisgreat.com/dodder/go/internal/0/dodder_env"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/repo_id"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	env_local "github.com/amarbel-llc/madder/go/pkgs/env_local"
	"github.com/amarbel-llc/piggy/go/pkgs/agent"
	"github.com/amarbel-llc/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/pool"
)

type Genesis struct {
	env_repo.BigBang
	LocalWorkingCopy
	BlobStore
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

	flagSet.BoolVar(
		&cmd.BigBang.YinDefault,
		"yin-default",
		false,
		"Use the embedded default zettel id left parts when -yin is unset",
	)

	flagSet.BoolVar(
		&cmd.BigBang.YangDefault,
		"yang-default",
		false,
		"Use the embedded default zettel id right parts when -yang is unset",
	)

	cmd.BigBang.SetDefaults()

	cmd.BigBang.GenesisConfig.Blob.SetFlagDefinitions(flagSet)

	cmd.BigBang.TypedBlobStoreConfig.Blob.SetFlagDefinitions(flagSet)

	flagSet.Var(
		getFlagValuePrivateKey(&cmd.BigBang.PrivateKey),
		"private_key",
		"pre-existing private key markl.Id (use info-ssh_agent to list keys)",
	)

	flagSet.Var(
		cmd.BlobStore.GetFlagValueBlobIds(&cmd.BlobStoreId),
		"blob_store-id",
		"The name of the existing madder blob store to use",
	)

	flagSet.BoolVar(
		&cmd.BigBang.ExcludeDefaultPandocTools,
		"exclude-default-pandoc-tools",
		false,
		"Exclude pandoc Lua filters and defaults from the default type",
	)

	flagSet.BoolVar(
		&cmd.BigBang.ExcludeDefaultType,
		"exclude-default-type",
		cmd.BigBang.ExcludeDefaultType,
		"Skip genesis-creating a default type; the repo starts with no default type, like a workspace repo or clone",
	)

	flagSet.BoolVar(
		&cmd.BigBang.IncludeBuiltinActionableTypes,
		"include-builtin-actionable-types",
		false,
		"Commit !task and !chore built-in types with status/priority/due fields and yq-based reader/writer scripts",
	)
}

// SetLocationFromPositionalRequired pops the required new-repo location
// positional and parses it into config.RepoId via the shared *Config
// pointer (FromAny returns a copy, so the mutation must go through the
// pointer). A non-auto config.RepoId means -repo_id / DODDER_REPO_ID was
// set, and under FDR-0021 T3-C those address an EXISTING repo rather than
// name a new one — so reject and point the caller at the positional.
func (cmd Genesis) SetLocationFromPositionalRequired(
	req command.Request,
	argName string,
) {
	config, ok := req.Utility.GetConfigAny().(*repo_config_cli.Config)
	if !ok {
		req.Cancel(
			errors.ErrorWithStackf(
				"expected *repo_config_cli.Config, got %T",
				req.Utility.GetConfigAny(),
			),
		)
		return
	}

	if !repo_id.IsAuto(config.RepoId) {
		req.Cancel(
			errors.BadRequestf(
				"-repo_id / DODDER_REPO_ID addresses an existing repo and "+
					"cannot name a new one; pass the new repo's location as "+
					"the %q positional instead",
				argName,
			),
		)
		return
	}

	if err := config.RepoId.Set(req.PopArg(argName)); err != nil {
		req.Cancel(err)
		return
	}
}

func (cmd Genesis) OnTheFirstDay(
	req command.Request,
) *local_working_copy.Repo {
	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())
	envUI := env_ui.Make(
		req,
		config,
		config.Debug,
		env_ui.Options{},
	)

	if err := repo_id.CheckSupported(config.RepoId); err != nil {
		envUI.Cancel(err)
	}

	ownDir := env_dir.MakeDefaultAndInitialize(
		req,
		dodder_env.XDGUtilityName,
		config.Debug,
		repo_id.EffectiveId(config.RepoId),
	)

	madderDir := env_dir.MakeDefaultAndInitialize(
		req,
		XDGUtilityNameMadder,
		config.Debug,
		repo_id.EffectiveId(config.RepoId),
	)

	var envRepo env_repo.Env

	options := env_repo.Options{
		BasePath:                config.BasePath,
		PermitNoDodderDirectory: true,
	}

	{
		var err error

		if envRepo, err = env_repo.Make(
			env_local.Make(envUI, ownDir),
			env_local.Make(envUI, madderDir),
			options,
		); err != nil {
			envUI.Cancel(err)
		}
	}

	envRepo.Genesis(cmd.BigBang)

	return local_working_copy.Genesis(cmd.BigBang, envRepo)
}

func getFlagValuePrivateKey(
	privateKey *markl.Id,
) interfaces.FlagValue {
	return command.FlagValueCompleter{
		FlagValue: privateKey,
		FuncCompleter: func(
			_ command.Request,
			envLocal env_local.Env,
			_ command.CommandLineInput,
		) {
			bufferedWriter, repool := pool.GetBufferedWriter(
				envLocal.GetUIFile(),
			)
			defer repool()

			defer errors.ContextMustFlush(envLocal, bufferedWriter)

			if keys, err := agent.DiscoverSSHAgentEd25519KeysVerbose(); err == nil {
				writeDiscoveredKeys(bufferedWriter, keys)
			}

			if keys, err := agent.DiscoverSSHAgentECDHKeysVerbose(); err == nil {
				writeDiscoveredKeys(bufferedWriter, keys)
			}
		},
	}
}

func writeDiscoveredKeys(
	bufferedWriter *bufio.Writer,
	keys []agent.DiscoveredKey,
) {
	for _, dk := range keys {
		text, err := dk.Id.MarshalText()
		if err != nil {
			continue
		}

		bufferedWriter.Write(text)
		bufferedWriter.WriteByte('\t')
		bufferedWriter.WriteString(dk.KeyType)

		if dk.Comment != "" {
			bufferedWriter.WriteString(": ")
			bufferedWriter.WriteString(dk.Comment)
		}

		bufferedWriter.WriteByte('\n')
	}
}
