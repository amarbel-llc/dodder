package commands_madder

import (
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/delta/xdg"
	"code.linenisgreat.com/dodder/go/internal/echo/directory_layout"
	"code.linenisgreat.com/dodder/go/internal/golf/blob_store_configs"
	"code.linenisgreat.com/dodder/go/internal/india/blob_stores"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/kilo/command_components_madder"
)

func init() {
	// TODO rename to repo-info
	utility.AddCmd("info-repo", &InfoRepo{})
}

type InfoRepo struct {
	command_components_madder.EnvBlobStore
	command_components_madder.BlobStore
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
