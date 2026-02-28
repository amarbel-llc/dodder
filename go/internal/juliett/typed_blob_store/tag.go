package typed_blob_store

import (
	"context"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/env_repo"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/blob_library"
	"code.linenisgreat.com/dodder/go/internal/hotel/sku_lua"
	"code.linenisgreat.com/dodder/go/internal/hotel/sku_wasm"
	"code.linenisgreat.com/dodder/go/internal/india/env_lua"
	"code.linenisgreat.com/dodder/go/internal/india/tag_blobs"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/lua"
	"code.linenisgreat.com/dodder/go/lib/charlie/wasm"
	"code.linenisgreat.com/dodder/go/lib/delta/toml"
)

type Tag struct {
	envRepo env_repo.Env
	envLua  env_lua.Env
	wasmRt  *wasm.Runtime
	toml_v0 domain_interfaces.TypedStore[tag_blobs.V0, *tag_blobs.V0]
	toml_v1 domain_interfaces.TypedStore[tag_blobs.TomlV1, *tag_blobs.TomlV1]
	lua_v1  domain_interfaces.TypedStore[tag_blobs.LuaV1, *tag_blobs.LuaV1]
	lua_v2  domain_interfaces.TypedStore[tag_blobs.LuaV2, *tag_blobs.LuaV2]
	wasm_v1 domain_interfaces.TypedStore[tag_blobs.WasmV1, *tag_blobs.WasmV1]
}

func MakeTagStore(
	envRepo env_repo.Env,
	envLua env_lua.Env,
	wasmRt *wasm.Runtime,
) Tag {
	return Tag{
		envRepo: envRepo,
		envLua:  envLua,
		wasmRt:  wasmRt,
		toml_v0: blob_library.MakeBlobStore(
			envRepo,
			blob_library.MakeBlobFormat(
				toml.MakeTomlDecoderIgnoreTomlErrors[tag_blobs.V0](
					envRepo.GetDefaultBlobStore(),
				),
				toml.TomlBlobEncoder[tag_blobs.V0, *tag_blobs.V0]{},
				envRepo.GetDefaultBlobStore(),
			),
			func(a *tag_blobs.V0) {
				a.Reset()
			},
		),
		toml_v1: blob_library.MakeBlobStore(
			envRepo,
			blob_library.MakeBlobFormat(
				toml.MakeTomlDecoderIgnoreTomlErrors[tag_blobs.TomlV1](
					envRepo.GetDefaultBlobStore(),
				),
				toml.TomlBlobEncoder[tag_blobs.TomlV1, *tag_blobs.TomlV1]{},
				envRepo.GetDefaultBlobStore(),
			),
			func(a *tag_blobs.TomlV1) {
				a.Reset()
			},
		),
		lua_v1: blob_library.MakeBlobStore(
			envRepo,
			blob_library.MakeBlobFormat[tag_blobs.LuaV1](
				nil,
				nil,
				envRepo.GetDefaultBlobStore(),
			),
			func(a *tag_blobs.LuaV1) {
			},
		),
		lua_v2: blob_library.MakeBlobStore(
			envRepo,
			blob_library.MakeBlobFormat[tag_blobs.LuaV2](
				nil,
				nil,
				envRepo.GetDefaultBlobStore(),
			),
			func(a *tag_blobs.LuaV2) {
			},
		),
		wasm_v1: blob_library.MakeBlobStore(
			envRepo,
			blob_library.MakeBlobFormat[tag_blobs.WasmV1](
				nil,
				nil,
				envRepo.GetDefaultBlobStore(),
			),
			func(a *tag_blobs.WasmV1) {
			},
		),
	}
}

// TODO check repool funcs
func (store Tag) GetBlob(
	object *sku.Transacted,
) (blobGeneric tag_blobs.Blob, repool interfaces.FuncRepool, err error) {
	tipe := object.GetType()
	blobId := object.GetBlobDigest()

	switch tipe.String() {
	case "", ids.TypeTomlTagV0:
		if blobGeneric, repool, err = store.toml_v0.GetBlob(blobId); err != nil {
			err = errors.Wrap(err)
			return blobGeneric, repool, err
		}

	case ids.TypeTomlTagV1:
		var blob *tag_blobs.TomlV1

		if blob, repool, err = store.toml_v1.GetBlob(blobId); err != nil {
			err = errors.Wrap(err)
			return blobGeneric, repool, err
		}

		luaVMPoolBuilder := store.envLua.MakeLuaVMPoolBuilder().WithApply(
			tag_blobs.MakeLuaSelfApplyV1(object),
		)

		var luaVMPool *lua.VMPool

		luaVMPoolBuilder.WithScript(blob.Filter)

		if luaVMPool, err = luaVMPoolBuilder.Build(); err != nil {
			err = errors.Wrap(err)
			return blobGeneric, repool, err
		}

		blob.LuaVMPoolV1 = sku_lua.MakeLuaVMPoolV1(luaVMPool, nil)
		blobGeneric = blob

	case ids.TypeLuaTagV1:
		// TODO try to repool things here

		var readCloser domain_interfaces.BlobReader

		if readCloser, err = store.envRepo.GetDefaultBlobStore().MakeBlobReader(
			blobId,
		); err != nil {
			err = errors.Wrap(err)
			return blobGeneric, repool, err
		}

		defer errors.DeferredCloser(&err, readCloser)

		luaVMPoolBuilder := store.envLua.MakeLuaVMPoolBuilder().WithApply(
			tag_blobs.MakeLuaSelfApplyV1(object),
		)

		var luaVMPool *lua.VMPool

		luaVMPoolBuilder.WithReader(readCloser)

		if luaVMPool, err = luaVMPoolBuilder.Build(); err != nil {
			err = errors.Wrap(err)
			return blobGeneric, repool, err
		}

		blobGeneric = &tag_blobs.LuaV1{
			LuaVMPoolV1: sku_lua.MakeLuaVMPoolV1(luaVMPool, nil),
		}

	case ids.TypeLuaTagV2:
		// TODO try to repool things here

		var readCloser domain_interfaces.BlobReader

		if readCloser, err = store.envRepo.GetDefaultBlobStore().MakeBlobReader(blobId); err != nil {
			err = errors.Wrap(err)
			return blobGeneric, repool, err
		}

		defer errors.DeferredCloser(&err, readCloser)

		luaVMPoolBUilder := store.envLua.MakeLuaVMPoolBuilder().WithApply(
			tag_blobs.MakeLuaSelfApplyV2(object),
		)

		var luaVMPool *lua.VMPool

		luaVMPoolBUilder.WithReader(readCloser)

		if luaVMPool, err = luaVMPoolBUilder.Build(); err != nil {
			err = errors.Wrap(err)
			return blobGeneric, repool, err
		}

		blobGeneric = &tag_blobs.LuaV2{
			LuaVMPoolV2: sku_lua.MakeLuaVMPoolV2(luaVMPool, nil),
		}

	case ids.TypeWasmTagV1:
		if store.wasmRt == nil {
			err = errors.ErrorWithStackf("WASM runtime not initialized")
			return blobGeneric, repool, err
		}

		var readCloser domain_interfaces.BlobReader

		if readCloser, err = store.envRepo.GetDefaultBlobStore().MakeBlobReader(
			blobId,
		); err != nil {
			err = errors.Wrap(err)
			return blobGeneric, repool, err
		}

		defer errors.DeferredCloser(&err, readCloser)

		ctx := context.Background()

		modulePool, buildErr := wasm.MakeModulePoolBuilder(store.wasmRt).
			WithReader(readCloser).
			Build(ctx)

		if buildErr != nil {
			err = errors.Wrap(buildErr)
			return blobGeneric, repool, err
		}

		blobGeneric = &tag_blobs.WasmV1{
			WasmVMPoolV1: sku_wasm.MakeWasmVMPoolV1(modulePool),
			Ctx:          ctx,
		}
	}

	return blobGeneric, repool, err
}
