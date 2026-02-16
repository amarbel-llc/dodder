package env_lua

import (
	"strings"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/lua"
	"code.linenisgreat.com/dodder/go/src/charlie/catgut"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/kilo/box_format"
)

// TODO extract all of these components into an env_lua

type Env interface {
	MakeLuaVMPoolBuilder() *lua.VMPoolBuilder
	GetSkuFromString(lv string) (sk *sku.Transacted, err error)
}

type env struct {
	envRepo      env_repo.Env
	objectStore  sku.RepoStore
	luaSkuFormat *box_format.BoxTransacted
}

func Make(
	envRepo env_repo.Env,
	objectStore sku.RepoStore,
	luaSkuFormat *box_format.BoxTransacted,
) *env {
	return &env{
		envRepo:      envRepo,
		objectStore:  objectStore,
		luaSkuFormat: luaSkuFormat,
	}
}

func (repo *env) MakeLuaVMPoolBuilder() *lua.VMPoolBuilder {
	return (&lua.VMPoolBuilder{}).WithSearcher(repo.luaSearcher)
}

func (s *env) luaSearcher(ls *lua.LState) int {
	lv := ls.ToString(1)
	ls.Pop(1)

	var err error

	if _, err = s.GetSkuFromString(lv); err != nil {
		ls.Push(lua.LString(err.Error()))
		return 1
	}
	ls.Push(ls.NewFunction(s.LuaRequire))

	return 1
}

func (s *env) GetSkuFromString(lv string) (object *sku.Transacted, err error) {
	object, _ = sku.GetTransactedPool().GetWithRepool()

	defer func() {
		if err != nil {
			return
		}

		if err = s.objectStore.ReadOneInto(object.GetObjectId(), object); err != nil {
			err = errors.Wrap(err)
			return
		}
	}()

	if err = ids.SetOnlyNotUnknownGenre(&object.ObjectId, lv); err == nil {
		return object, err
	}

	rb := catgut.MakeRingBuffer(strings.NewReader(lv), 0)

	if _, err = s.luaSkuFormat.ReadStringFormat(
		object,
		catgut.MakeRingBufferRuneScanner(rb),
	); err == nil {
		return object, err
	}

	return object, err
}

// TODO modify `package.loaded` to include variations of object id
func (s *env) LuaRequire(ls *lua.LState) int {
	// TODO handle second extra arg
	// TODO parse lv as object id / blob
	lv := ls.ToString(1)
	ls.Pop(1)

	var err error
	var object *sku.Transacted

	if object, err = s.GetSkuFromString(lv); err != nil {
		panic(err)
		// ls.Push(lua.LString(err.Error()))
		// return 1
	}

	// object lifecycle managed by caller

	if err = s.objectStore.ReadOneInto(object.GetObjectId(), object); err != nil {
		panic(err)
	}

	var ar domain_interfaces.BlobReader

	if ar, err = s.envRepo.GetDefaultBlobStore().MakeBlobReader(
		object.GetBlobDigest(),
	); err != nil {
		panic(err)
	}

	defer errors.DeferredCloser(&err, ar)

	var compiled *lua.FunctionProto

	if compiled, err = lua.CompileReader(ar); err != nil {
		panic(err)
	}

	ls.Push(ls.NewFunctionFromProto(compiled))

	if err = ls.PCall(0, 1, nil); err != nil {
		panic(err)
	}

	return 1
}
