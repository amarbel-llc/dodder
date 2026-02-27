package command_components_madder

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/alfa/pool"
	"code.linenisgreat.com/dodder/go/internal/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/internal/india/env_local"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
)

type Complete struct {
	EnvBlobStore
}

func (cmd Complete) GetFlagValueBlobIds(
	blobStoreId *blob_store_id.Id,
) interfaces.FlagValue {
	return command.FlagValueCompleter{
		FlagValue: blobStoreId,
		FuncCompleter: func(
			req command.Request,
			envLocal env_local.Env,
			commandLine command.CommandLineInput,
		) {
			envBlobStore := cmd.MakeEnvBlobStore(req)
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
