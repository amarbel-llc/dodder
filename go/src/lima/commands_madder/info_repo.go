package commands_madder

import (
	"strings"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/delta/xdg"
	"code.linenisgreat.com/dodder/go/src/echo/directory_layout"
	"code.linenisgreat.com/dodder/go/src/golf/blob_store_configs"
	"code.linenisgreat.com/dodder/go/src/india/blob_stores"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"code.linenisgreat.com/dodder/go/src/kilo/command_components_madder"
)

func init() {
	// TODO rename to repo-info
	utility.AddCmd("info-repo", &InfoRepo{})
}

type InfoRepo struct {
	command_components_madder.EnvBlobStore
	command_components_madder.BlobStoreConfig
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

	for _, key := range keys {
		switch strings.ToLower(key) {
		default:
			errors.ContextCancelWithBadRequestf(
				env,
				"unsupported info key: %q",
				key,
			)

		case "config-immutable":
			if _, err := blob_store_configs.Coder.EncodeTo(
				&blobStoreConfig,
				env.GetUIFile(),
			); err != nil {
				env.Cancel(err)
			}

		case "name":
			// TODO

			// TODO switch to `blob_stores.N.compression_type`
		case "compression-type":
			blobIOWrapper := blobStore.GetBlobIOWrapper()

			// TODO read default blob store and expose config
			env.GetUI().Print(
				blobIOWrapper.GetBlobCompression(),
			)

			// TODO switch to `blob_stores.N.age_encryption`
		case "blob_stores-0-encryption":
			blobIOWrapper := blobStore.GetBlobIOWrapper()

			env.GetUI().Print(
				blobIOWrapper.GetBlobEncryption().StringWithFormat(),
			)

		case "blob_stores-0-config-path":
			env.GetUI().Print(
				directory_layout.GetDefaultBlobStore(env).GetConfig(),
			)

		case "blob_stores-0-config":
			blobStoreConfig := blobStore.ConfigNamed.Config

			if err := cmd.PrintBlobStoreConfig(
				env,
				&blob_store_configs.TypedConfig{
					Type: blobStoreConfig.Type,
					Blob: blobStoreConfig.Blob,
				},
				env.GetUIFile(),
			); err != nil {
				env.Cancel(err)
				return
			}

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
		}
	}
}
