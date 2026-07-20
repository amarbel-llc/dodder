package sku_lua

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/lua"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/pool"
)

type LuaVMV1 struct {
	lua.LValue
	*lua.VM
	TablePool LuaTablePoolV1
	Selbst    *sku.Transacted
}

func PushTopFuncV1(
	lvm LuaVMPoolV1,
	args []string,
) (vm *LuaVMV1, argsOut []string, err error) {
	vm, _ = lvm.GetWithRepool() //repool:owned

	vm.LValue = vm.Top

	var f *lua.LFunction

	if f, argsOut, err = vm.GetTopFunctionOrFunctionNamedError(
		args,
	); err != nil {
		err = errors.Wrap(err)
		return vm, argsOut, err
	}

	vm.Push(f)

	return vm, argsOut, err
}

type (
	LuaVMPoolV1    = interfaces.PoolPtr[LuaVMV1, *LuaVMV1]
	LuaTablePoolV1 = interfaces.PoolPtr[LuaTableV1, *LuaTableV1]
)

func MakeLuaVMPoolV1(vmPool *lua.VMPool, self *sku.Transacted) LuaVMPoolV1 {
	return pool.Make(
		func() (out *LuaVMV1) {
			vm, _ := vmPool.PoolPtr.GetWithRepool() //repool:owned

			out = &LuaVMV1{
				VM:        vm,
				TablePool: MakeLuaTablePoolV1(vm),
				Selbst:    self,
			}

			return out
		},
		nil,
	)
}

func MakeLuaTablePoolV1(vm *lua.VM) LuaTablePoolV1 {
	return pool.Make(
		func() (table *LuaTableV1) {
			transacted, _ := vm.PoolPtr.GetWithRepool()   //repool:owned
			tags, _ := vm.PoolPtr.GetWithRepool()         //repool:owned
			tagsImplicit, _ := vm.PoolPtr.GetWithRepool() //repool:owned
			fields, _ := vm.PoolPtr.GetWithRepool()       //repool:owned

			table = &LuaTableV1{
				Transacted:   transacted,
				Tags:         tags,
				TagsImplicit: tagsImplicit,
				Fields:       fields,
			}

			vm.SetField(table.Transacted, "Etiketten", table.Tags)
			vm.SetField(
				table.Transacted,
				"EtikettenImplicit",
				table.TagsImplicit,
			)
			vm.SetField(table.Transacted, "Fields", table.Fields)

			return table
		},
		func(t *LuaTableV1) {
			lua.ClearTable(vm.LState, t.Tags)
			lua.ClearTable(vm.LState, t.TagsImplicit)
			lua.ClearTable(vm.LState, t.Fields)
		},
	)
}
