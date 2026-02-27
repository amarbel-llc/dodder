package typed_blob_store

import (
	"code.linenisgreat.com/dodder/go/internal/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/internal/kilo/box_format"
	"code.linenisgreat.com/dodder/go/internal/lima/env_lua"
	"code.linenisgreat.com/dodder/go/internal/lima/inventory_list_coders"
	"code.linenisgreat.com/dodder/go/internal/lima/type_blobs"
)

type Stores struct {
	InventoryList inventory_list_coders.Closet
	Repo          RepoStore
	Type          type_blobs.Coder
	Tag           Tag
}

func MakeStores(
	envRepo env_repo.Env,
	envLua env_lua.Env,
	boxFormat *box_format.BoxTransacted,
) Stores {
	return Stores{
		InventoryList: inventory_list_coders.MakeCloset(
			envRepo,
			boxFormat,
		),
		Tag:  MakeTagStore(envRepo, envLua),
		Repo: MakeRepoStore(envRepo),
		Type: type_blobs.MakeTypeStore(envRepo),
	}
}

// func (stores Stores) GetTypeV1() interfaces.TypedStore[type_blobs.TomlV1, *type_blobs.TomlV1] {
// 	return stores.Type.toml_v1
// }
