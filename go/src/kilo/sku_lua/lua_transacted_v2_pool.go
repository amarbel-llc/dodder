package sku_lua

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/bravo/lua"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

type LuaVMV2 struct {
	lua.LValue
	*lua.VM
	TablePool LuaTablePoolV2
	Selbst    *sku.Transacted
}

func PushTopFuncV2(
	luaVMPool LuaVMPoolV2,
	args []string,
) (vm *LuaVMV2, argsOut []string, err error) {
	vm, _ = luaVMPool.GetWithRepool()

	vm.LValue = vm.Top

	var luaFunction *lua.LFunction

	if luaFunction, argsOut, err = vm.GetTopFunctionOrFunctionNamedError(
		args,
	); err != nil {
		err = errors.Wrap(err)
		return vm, argsOut, err
	}

	vm.Push(luaFunction)

	return vm, argsOut, err
}

type (
	LuaVMPoolV2    = interfaces.PoolPtr[LuaVMV2, *LuaVMV2]
	LuaTablePoolV2 = interfaces.PoolPtr[LuaTableV2, *LuaTableV2]
)

func MakeLuaVMPoolV2(luaVMPool *lua.VMPool, self *sku.Transacted) LuaVMPoolV2 {
	return pool.Make(
		func() (out *LuaVMV2) {
			vm, _ := luaVMPool.PoolPtr.GetWithRepool()

			out = &LuaVMV2{
				VM:        vm,
				TablePool: MakeLuaTablePoolV2(vm),
				Selbst:    self,
			}

			return out
		},
		nil,
	)
}

func MakeLuaTablePoolV2(vm *lua.VM) LuaTablePoolV2 {
	return pool.Make(
		func() (t *LuaTableV2) {
			transacted, _ := vm.PoolPtr.GetWithRepool()
			tags, _ := vm.PoolPtr.GetWithRepool()
			tagsImplicit, _ := vm.PoolPtr.GetWithRepool()

			t = &LuaTableV2{
				Transacted:   transacted,
				Tags:         tags,
				TagsImplicit: tagsImplicit,
			}

			vm.SetField(t.Transacted, "Tags", t.Tags)
			vm.SetField(t.Transacted, "TagsImplicit", t.TagsImplicit)

			return t
		},
		func(t *LuaTableV2) {
			lua.ClearTable(vm.LState, t.Tags)
			lua.ClearTable(vm.LState, t.TagsImplicit)
		},
	)
}
