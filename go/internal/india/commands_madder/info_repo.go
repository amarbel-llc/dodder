package commands_madder

import (
	"strings"

	"code.linenisgreat.com/dodder/go/internal/bravo/directory_layout"
	"code.linenisgreat.com/dodder/go/internal/delta/blob_store_configs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/hotel/command_components_madder"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/echo/xdg"
)

func init() {
	// TODO rename to repo-info
	utility.AddCmd("info-repo", &InfoRepo{})
}

type InfoRepo struct {
	command_components_madder.EnvBlobStore
	command_components_madder.BlobStore
}

var _ command.CommandWithArgs = (*InfoRepo)(nil)

func (cmd *InfoRepo) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{
			{
				Name:        "store-id",
				Description: "blob store to query (defaults to default store)",
			},
			{
				Name:        "keys",
				Description: "config keys to display (defaults to config-immutable)",
				Variadic:    true,
			},
		},
	}}
}

func (cmd InfoRepo) GetDescription() command.Description {
	return command.Description{
		Short: "display blob store configuration",
		Long: "Show the configuration of a blob store in hyphence format. " +
			"With no arguments, shows the default store's immutable config. " +
			"Accepts a store ID and one or more config keys: " +
			"config-immutable (default), config-path, dir-blob_stores, " +
			"xdg, or any key from the store's typed config.",
	}
}

func (cmd InfoRepo) Run(req command.Request) {
	env := cmd.MakeEnvBlobStore(req)

	var blobStore blob_stores.BlobStoreInitialized
	var keys []string

	switch req.RemainingArgCount() {
	case 0:
		blobStore = env.GetDefaultBlobStore()
		keys = []string{"config-immutable"}

	case 1:
		blobStore = env.GetDefaultBlobStore()
		keys = []string{req.PopArg("blob store config key")}

	case 2:
		blobStoreIndex := req.PopArg("blob store index")
		blobStore = cmd.MakeBlobStoreFromIdString(env, blobStoreIndex)
		keys = []string{req.PopArg("blob store config key")}

	default:
		blobStoreIndex := req.PopArg("blob store index")
		blobStore = cmd.MakeBlobStoreFromIdString(env, blobStoreIndex)
		keys = req.PopArgs()
	}

	blobStoreConfig := blobStore.Config
	configKVs := blob_store_configs.ConfigKeyValues(blobStoreConfig.Blob)

	for _, key := range keys {
		switch strings.ToLower(key) {
		case "config-immutable":
			if _, err := blob_store_configs.Coder.EncodeTo(
				&blobStoreConfig,
				env.GetUIFile(),
			); err != nil {
				env.Cancel(err)
			}

		case "config-path":
			env.GetUI().Print(
				directory_layout.GetDefaultBlobStore(env).GetConfig(),
			)

		case "dir-blob_stores":
			env.GetUI().Print(env.MakePathBlobStore())

		case "xdg":
			ecksDeeGee := env.GetXDG()

			dotenv := xdg.Dotenv{
				XDG: &ecksDeeGee,
			}

			if _, err := dotenv.WriteTo(env.GetUIFile()); err != nil {
				env.Cancel(err)
			}

		default:
			value, ok := configKVs[key]
			if !ok {
				availableKeys := blob_store_configs.ConfigKeyNames(
					blobStoreConfig.Blob,
				)

				errors.ContextCancelWithBadRequestf(
					env,
					"unsupported info key: %q\navailable keys: %s",
					key,
					strings.Join(availableKeys, ", "),
				)

				return
			}

			env.GetUI().Print(value)
		}
	}
}
