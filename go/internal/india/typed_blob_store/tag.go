package typed_blob_store

import (
	"io"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/blob_library"
	"code.linenisgreat.com/dodder/go/internal/golf/sku_lua"
	"code.linenisgreat.com/dodder/go/internal/hotel/env_lua"
	"code.linenisgreat.com/dodder/go/internal/hotel/tag_blobs"
	"code.linenisgreat.com/dodder/go/lib/alfa/lua"
	"github.com/amarbel-llc/madder/go/pkgs/hyphence"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

type Tag struct {
	envRepo env_repo.Env
	envLua  env_lua.Env
	toml_v0 mad_domain_interfaces.TypedStore[tag_blobs.V0, *tag_blobs.V0]
	toml_v1 mad_domain_interfaces.TypedStore[tag_blobs.TomlV1, *tag_blobs.TomlV1]
	lua_v1  mad_domain_interfaces.TypedStore[tag_blobs.LuaV1, *tag_blobs.LuaV1]
	lua_v2  mad_domain_interfaces.TypedStore[tag_blobs.LuaV2, *tag_blobs.LuaV2]
}

func MakeTagStore(
	envRepo env_repo.Env,
	envLua env_lua.Env,
) Tag {
	return Tag{
		envRepo: envRepo,
		envLua:  envLua,
		toml_v0: blob_library.MakeBlobStore(
			envRepo,
			blob_library.MakeBlobFormat(
				noopBlobDecoder[tag_blobs.V0, *tag_blobs.V0]{},
				noopBlobEncoder[tag_blobs.V0, *tag_blobs.V0]{},
				envRepo.GetDefaultBlobStore(),
			),
			func(a *tag_blobs.V0) {
				a.Reset()
			},
		),
		toml_v1: blob_library.MakeBlobStore(
			envRepo,
			blob_library.MakeBlobFormat(
				hyphence.TommyBlobDecoder[tag_blobs.TomlV1, *tag_blobs.TomlV1]{
					Decode: func(b []byte) (tag_blobs.TomlV1, error) {
						doc, err := tag_blobs.DecodeTomlV1(b)
						if err != nil {
							return tag_blobs.TomlV1{}, err
						}
						return *doc.Data(), nil
					},
				},
				hyphence.TommyBlobEncoder[tag_blobs.TomlV1, *tag_blobs.TomlV1]{
					Encode: func(v tag_blobs.TomlV1) ([]byte, error) {
						doc, err := tag_blobs.DecodeTomlV1(nil)
						if err != nil {
							return nil, err
						}
						*doc.Data() = v
						return doc.Encode()
					},
				},
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

		var readCloser mad_domain_interfaces.BlobReader

		if readCloser, err = store.envRepo.GetReadBlobStore().MakeBlobReader(
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

		var readCloser mad_domain_interfaces.BlobReader

		if readCloser, err = store.envRepo.GetReadBlobStore().MakeBlobReader(blobId); err != nil {
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
	}

	return blobGeneric, repool, err
}

type noopBlobDecoder[
	BLOB any,
	BLOB_PTR interfaces.Ptr[BLOB],
] struct{}

func (noopBlobDecoder[BLOB, BLOB_PTR]) DecodeFrom(
	blob BLOB_PTR,
	reader io.Reader,
) (n int64, err error) {
	n, err = io.Copy(io.Discard, reader)
	if err != nil {
		err = errors.Wrap(err)
	}

	return
}

type noopBlobEncoder[
	BLOB any,
	BLOB_PTR interfaces.Ptr[BLOB],
] struct{}

func (noopBlobEncoder[BLOB, BLOB_PTR]) EncodeTo(
	blob BLOB_PTR,
	writer io.Writer,
) (n int64, err error) {
	return
}
