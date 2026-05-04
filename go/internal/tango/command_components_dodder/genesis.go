package command_components_dodder

import (
	"bufio"

	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/charlie/env_local"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/alfa/pool"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
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
		&cmd.BigBang.IncludeDefaultPandocTools,
		"include-default-pandoc-tools",
		false,
		"Include pandoc Lua filters and defaults as blob references on the default type",
	)

	flagSet.BoolVar(
		&cmd.BigBang.IncludeBuiltinActionableTypes,
		"include-builtin-actionable-types",
		false,
		"Commit !task and !chore built-in types with status/priority/due fields and yq-based reader/writer scripts",
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
		config.RepoId,
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

			if keys, err := markl.DiscoverSSHAgentEd25519KeysVerbose(); err == nil {
				writeDiscoveredKeys(bufferedWriter, keys)
			}

			if keys, err := markl.DiscoverSSHAgentECDHKeysVerbose(); err == nil {
				writeDiscoveredKeys(bufferedWriter, keys)
			}
		},
	}
}

func writeDiscoveredKeys(
	bufferedWriter *bufio.Writer,
	keys []markl.DiscoveredKey,
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
