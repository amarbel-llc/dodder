package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/tango/repo"
	"code.linenisgreat.com/dodder/go/internal/uniform/remote_transfer"
)

func (local *Repo) MakeImporter(
	options repo.ImporterOptions,
	storeOptions sku.StoreOptions,
) repo.Importer {
	store := local.GetStore()

	return remote_transfer.Make(
		options,
		storeOptions,
		store.GetEnvRepo(),
		store.GetTypedBlobStore().InventoryList,
		store.GetStreamIndex(),
		local.GetEnvWorkspace().GetStore(),
		store,
	)
}
