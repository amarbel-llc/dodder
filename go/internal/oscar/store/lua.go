package store

import (
	"io"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/sku_lua"
	"code.linenisgreat.com/dodder/go/internal/hotel/tag_blobs"
	"code.linenisgreat.com/dodder/go/lib/alfa/lua"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
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

	var readCloser mad_domain_interfaces.BlobReader

	if readCloser, err = store.GetEnvRepo().GetReadBlobStore().MakeBlobReader(sk.GetBlobDigest()); err != nil {
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
