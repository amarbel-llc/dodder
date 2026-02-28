package command_components_madder

import (
	"code.linenisgreat.com/dodder/go/internal/_/blob_store_id"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_local"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
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
