package commands_dodder

import (
	"sort"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/delta/blob_store_configs"
	"code.linenisgreat.com/dodder/go/internal/echo/genesis_configs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_local"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/echo/xdg"
)

func init() {
	// TODO rename to repo-info
	utility.AddCmd("info-repo", &InfoRepo{})
}

type InfoRepo struct {
	command_components_dodder.EnvRepo
}

var repoSpecialKeys = []string{
	"config-immutable",
	"id",
	"pubkey",
	"seckey",
	"store-version",
	"xdg",
}

func (cmd InfoRepo) Run(req command.Request) {
	args := req.PopArgs()
	env := cmd.MakeEnvRepo(req, false)

	configPublicTypedBlob := env.GetConfigPublic()
	configPublicBlob := configPublicTypedBlob.Blob

	configPrivateTypedBlob := env.GetConfigPrivate()
	configPrivateBlob := configPrivateTypedBlob.Blob

	defaultBlobStore := env.GetDefaultBlobStore()

	if len(args) == 0 {
		args = []string{"store-version"}
	}

	configKVs := blob_store_configs.ConfigKeyValues(
		defaultBlobStore.Config.Blob,
	)

	for _, arg := range args {
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
			env.GetUI().Print(configPublicBlob.GetRepoId())

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
				allKeys := allAvailableKeys(
					defaultBlobStore.Config.Blob,
				)

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

func allAvailableKeys(config blob_store_configs.Config) []string {
	configKeys := blob_store_configs.ConfigKeyNames(config)
	allKeys := make([]string, 0, len(repoSpecialKeys)+len(configKeys))
	allKeys = append(allKeys, repoSpecialKeys...)
	allKeys = append(allKeys, configKeys...)
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
	keys := allAvailableKeys(defaultBlobStore.Config.Blob)

	for _, key := range keys {
		envLocal.GetUI().Print(key)
	}
}
