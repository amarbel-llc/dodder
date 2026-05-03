package command_components_dodder

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/directory_layout"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_dir"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/charlie/env_local"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/lib/0/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/debug"
	"code.linenisgreat.com/dodder/go/lib/charlie/config_cli"
	"github.com/amarbel-llc/madder/go/pkgs/blob_store_configs"
	"github.com/amarbel-llc/madder/go/pkgs/blob_store_id"
	"github.com/amarbel-llc/madder/go/pkgs/blob_stores"
	"github.com/amarbel-llc/madder/go/pkgs/hyphence"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

type BlobStore struct{}

func (cmd *BlobStore) MakeBlobStoreFromIdOrConfigPath(
	envBlobStore env_repo.BlobStoreEnv,
	basePath string,
	blobStoreIndexOrConfigPath string,
) (blobStore blob_stores.BlobStoreInitialized) {
	if blobStoreIndexOrConfigPath == "" {
		goto tryDefaultBlobStore
	}

	{
		configPath := blobStoreIndexOrConfigPath
		var typedConfig blob_store_configs.TypedConfig

		{
			var err error

			if typedConfig, err = hyphence.DecodeFromFile(
				blob_store_configs.Coder,
				configPath,
			); err != nil {
				if errors.IsNotExist(err) {
					err = nil
					goto tryBlobStoreId
				} else {
					envBlobStore.Cancel(err)
					return blobStore
				}
			}
		}

		blobStore.Config = typedConfig

		blobStore.Path = directory_layout.GetBlobStorePathForCustomPath(
			blobStoreIndexOrConfigPath,
			basePath,
			blobStoreIndexOrConfigPath,
		)

		{
			var err error

			if blobStore.BlobStore, err = blob_stores.MakeBlobStore(
				envBlobStore,
				blobStore.ConfigNamed,
				nil,
			); err != nil {
				envBlobStore.Cancel(err)
				return blobStore
			}
		}

		return blobStore
	}

tryBlobStoreId:
	return cmd.MakeBlobStoreFromIdString(envBlobStore, blobStoreIndexOrConfigPath)

tryDefaultBlobStore:
	return envBlobStore.GetDefaultBlobStore()
}

func (cmd *BlobStore) MakeBlobStoreFromIdString(
	envBlobStore env_repo.BlobStoreEnv,
	blobStoreIdString string,
) (blobStore blob_stores.BlobStoreInitialized) {
	var blobStoreId blob_store_id.Id

	if err := blobStoreId.Set(blobStoreIdString); err != nil {
		envBlobStore.Cancel(err)
		return blobStore
	}

	return envBlobStore.GetBlobStore(blobStoreId)
}

func (cmd BlobStore) GetFlagValueBlobIds(
	blobStoreId *blob_store_id.Id,
) interfaces.FlagValue {
	return command.FlagValueCompleter{
		FlagValue: blobStoreId,
		FuncCompleter: func(
			req command.Request,
			envLocal env_local.Env,
			commandLine command.CommandLineInput,
		) {
			envBlobStore := cmd.makeEnvBlobStore(req)
			blobStoresAll := envBlobStore.GetBlobStores()

			bufferedWriter, repool := pool.GetBufferedWriter(
				envLocal.GetUIFile(),
			)
			defer repool()

			defer errors.ContextMustFlush(envLocal, bufferedWriter)

			for _, blobStore := range blobStoresAll {
				bufferedWriter.WriteString(blobStore.Path.GetId().String())
				bufferedWriter.WriteByte('\t')
				bufferedWriter.WriteString(blobStore.GetBlobStoreDescription())
				bufferedWriter.WriteByte('\n')
			}
		},
	}
}

func (cmd BlobStore) makeEnvBlobStore(
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
