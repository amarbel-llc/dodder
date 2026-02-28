package typed_blob_store

import (
	"code.linenisgreat.com/dodder/go/internal/golf/env_repo"
	"code.linenisgreat.com/dodder/go/internal/hotel/box_format"
	"code.linenisgreat.com/dodder/go/internal/hotel/type_blobs"
	"code.linenisgreat.com/dodder/go/internal/india/env_lua"
	"code.linenisgreat.com/dodder/go/internal/india/inventory_list_coders"
	"code.linenisgreat.com/dodder/go/lib/charlie/wasm"
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
	wasmRt *wasm.Runtime,
	boxFormat *box_format.BoxTransacted,
) Stores {
	return Stores{
		InventoryList: inventory_list_coders.MakeCloset(
			envRepo,
			boxFormat,
		),
		Tag:  MakeTagStore(envRepo, envLua, wasmRt),
		Repo: MakeRepoStore(envRepo),
		Type: type_blobs.MakeTypeStore(envRepo),
	}
}

// func (stores Stores) GetTypeV1() interfaces.TypedStore[type_blobs.TomlV1, *type_blobs.TomlV1] {
// 	return stores.Type.toml_v1
// }
