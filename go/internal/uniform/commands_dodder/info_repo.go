package commands_dodder

import (
	"sort"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/charlie/genesis_configs"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/echo/repo_identity"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/madder/go/pkgs/blob_store_configs"
	env_local "code.linenisgreat.com/madder/go/pkgs/env_local"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/xdg"
)

func init() {
	// TODO rename to repo-info
	utility.AddCmd("info-repo", &InfoRepo{})
}

type InfoRepo struct {
	command_components_dodder.EnvRepo
}

func (cmd InfoRepo) GetDescription() command.Description {
	return command.Description{
		Short: "display repository configuration",
	}
}

var repoSpecialKeys = []string{
	"config-immutable",
	"id",
	"pubkey",
	"repos",
	"seckey",
	"store-version",
	"xdg",
}

var _ command.CommandWithArgs = (*InfoRepo)(nil)

func (cmd *InfoRepo) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{{
			Name:        "keys",
			Description: "config keys to display (defaults to store-version)",
			Variadic:    true,
			EnumValues:  repoSpecialKeys,
		}},
	}}
}

func (cmd InfoRepo) Run(req command.Request) {
	args := req.PopArgs()

	if len(args) == 0 {
		args = []string{"store-version"}
	}

	// `repos` is a discovery listing across the active scope, not an
	// inspection of one repo, so it must work without an opened repo (e.g.
	// from a dir with no initialized repo). Peel it off and handle it before
	// MakeEnvRepo; the remaining keys still read the opened repo's config.
	var rest []string

	for _, arg := range args {
		if strings.ToLower(arg) == "repos" {
			cmd.printRepos(req)
		} else {
			rest = append(rest, arg)
		}
	}

	if len(rest) == 0 {
		return
	}

	env := cmd.MakeEnvRepo(req, false)

	configPublicTypedBlob := env.GetConfigPublic()
	configPublicBlob := configPublicTypedBlob.Blob

	configPrivateTypedBlob := env.GetConfigPrivate()
	configPrivateBlob := configPrivateTypedBlob.Blob

	defaultBlobStore := env.GetDefaultBlobStore()

	configKVs := blob_store_configs.ConfigKeyValues(
		defaultBlobStore.Config.Blob,
	)

	// Under the write_through multi default (FDR-0016 D1 / dodder#223),
	// the default store's own config is pure routing (mode + digest-pinned
	// member refs) and carries none of the storage-facing keys
	// (encryption, compression-type, hash_type-id, ...) these queries ask
	// about. Delegate: merge the write store's key-values underneath the
	// multi's own, so `info-repo encryption` keeps answering for the
	// store writes actually land in. The multi's own keys win on
	// collision.
	if multi, ok := defaultBlobStore.Config.Blob.(blob_store_configs.ConfigMulti); ok {
		if writeStoreId := multi.GetWriteStore(); !writeStoreId.IsEmpty() {
			writeStore := env.GetEnvBlobStore().GetBlobStore(writeStoreId)

			for key, value := range blob_store_configs.ConfigKeyValues(
				writeStore.Config.Blob,
			) {
				if _, exists := configKVs[key]; !exists {
					configKVs[key] = value
				}
			}
		}
	}

	for _, arg := range rest {
		switch strings.ToLower(arg) {
		case "config-immutable":
			if _, err := genesis_configs.CoderPublic.EncodeTo(
				&configPublicTypedBlob,
				env.GetUIFile(),
			); err != nil {
				env.Cancel(err)
			}

		case "store-version":
			env.GetUI().Print(configPublicBlob.GetStoreVersion())

		case "id":
			config := repo_config_cli.FromAny(req.Utility.GetConfigAny())

			env.GetUI().Print(
				repo_identity.Render(
					config.GetRepoId().String(),
					configPublicBlob.GetPublicKey(),
				),
			)

		case "pubkey":
			env.GetUI().Print(
				configPublicBlob.GetPublicKey().StringWithFormat(),
			)

		case "seckey":
			env.Cancel(errors.Err405MethodNotAllowed)

			env.GetUI().Print(
				configPrivateBlob.GetPrivateKey().StringWithFormat(),
			)

		case "xdg":
			exdg := env.GetXDG()

			dotenv := xdg.Dotenv{
				XDG: &exdg,
			}

			if _, err := dotenv.WriteTo(env.GetUIFile()); err != nil {
				env.Cancel(err)
			}

		default:
			value, ok := configKVs[arg]
			if !ok {
				allKeys := allAvailableKeys(configKVs)

				errors.ContextCancelWithBadRequestf(
					env,
					"unsupported info key: %q\navailable keys: %s",
					arg,
					strings.Join(allKeys, ", "),
				)

				return
			}

			env.GetUI().Print(value)
		}
	}
}

// printRepos lists the repos addressable from here, one -repo_id spelling
// per line (`.name` for cwd repos, `name` for user repos), across both
// scopes. It uses a repo-less UI (like info) so it works even when the
// current directory has no initialized repo.
func (cmd InfoRepo) printRepos(req command.Request) {
	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())
	ui := env_ui.Make(req, config, config.Debug, env_ui.Options{})

	repos, err := listScopedRepos(req)
	if err != nil {
		ui.Cancel(err)
		return
	}

	for _, repo := range repos {
		ui.GetUI().Print(repo.Spelling())
	}
}

// allAvailableKeys takes the already-merged key-values map (the default
// store's own keys plus, for a multi default, its write store's — see
// the delegation in run) so the error listing matches what is actually
// queryable.
func allAvailableKeys(configKVs map[string]string) []string {
	allKeys := make([]string, 0, len(repoSpecialKeys)+len(configKVs))
	allKeys = append(allKeys, repoSpecialKeys...)

	for key := range configKVs {
		allKeys = append(allKeys, key)
	}

	sort.Strings(allKeys)

	return allKeys
}

func (cmd InfoRepo) Complete(
	req command.Request,
	envLocal env_local.Env,
	_ command.CommandLineInput,
) {
	env := cmd.MakeEnvRepo(req, false)
	defaultBlobStore := env.GetDefaultBlobStore()

	configKVs := blob_store_configs.ConfigKeyValues(
		defaultBlobStore.Config.Blob,
	)

	// Mirror run's multi delegation so completion offers the same keys
	// the query path answers.
	if multi, ok := defaultBlobStore.Config.Blob.(blob_store_configs.ConfigMulti); ok {
		if writeStoreId := multi.GetWriteStore(); !writeStoreId.IsEmpty() {
			writeStore := env.GetEnvBlobStore().GetBlobStore(writeStoreId)

			for key, value := range blob_store_configs.ConfigKeyValues(
				writeStore.Config.Blob,
			) {
				if _, exists := configKVs[key]; !exists {
					configKVs[key] = value
				}
			}
		}
	}

	for _, key := range allAvailableKeys(configKVs) {
		envLocal.GetUI().Print(key)
	}
}
