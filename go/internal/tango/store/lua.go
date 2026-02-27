package store

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/internal/lima/sku_lua"
	"code.linenisgreat.com/dodder/go/internal/mike/tag_blobs"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/lua"
)

func (store *Store) MakeLuaVMPoolV1WithSku(
	sk *sku.Transacted,
) (lvp sku_lua.LuaVMPoolV1, err error) {
	if sk.GetType().String() != "lua" {
		err = errors.ErrorWithStackf(
			"unsupported typ: %s, Sku: %s",
			sk.GetType(),
			sk,
		)
		return lvp, err
	}

	var readCloser domain_interfaces.BlobReader

	if readCloser, err = store.GetEnvRepo().GetDefaultBlobStore().MakeBlobReader(sk.GetBlobDigest()); err != nil {
		err = errors.Wrap(err)
		return lvp, err
	}

	defer errors.DeferredCloser(&err, readCloser)

	if lvp, err = store.MakeLuaVMPoolWithReader(sk, readCloser); err != nil {
		err = errors.Wrap(err)
		return lvp, err
	}

	return lvp, err
}

func (store *Store) MakeLuaVMPoolV1(
	self *sku.Transacted,
	script string,
) (vp sku_lua.LuaVMPoolV1, err error) {
	b := store.envLua.MakeLuaVMPoolBuilder().
		WithScript(script).
		WithApply(tag_blobs.MakeLuaSelfApplyV1(self))

	var lvmp *lua.VMPool

	if lvmp, err = b.Build(); err != nil {
		err = errors.Wrap(err)
		return vp, err
	}

	vp = sku_lua.MakeLuaVMPoolV1(lvmp, self)

	return vp, err
}

func (store *Store) MakeLuaVMPoolWithReader(
	selbst *sku.Transacted,
	r io.Reader,
) (vp sku_lua.LuaVMPoolV1, err error) {
	b := store.envLua.MakeLuaVMPoolBuilder().
		WithReader(r).
		WithApply(tag_blobs.MakeLuaSelfApplyV1(selbst))

	var lvmp *lua.VMPool

	if lvmp, err = b.Build(); err != nil {
		err = errors.Wrap(err)
		return vp, err
	}

	vp = sku_lua.MakeLuaVMPoolV1(lvmp, selbst)

	return vp, err
}
