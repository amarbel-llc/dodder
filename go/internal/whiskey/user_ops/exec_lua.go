package user_ops

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/lua"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku_lua"
	"code.linenisgreat.com/dodder/go/internal/victor/local_working_copy"
)

type ExecLua struct {
	*local_working_copy.Repo
}

func (u ExecLua) Run(sk *sku.Transacted, args ...string) (err error) {
	var lvp sku_lua.LuaVMPoolV1

	if lvp, err = u.GetStore().MakeLuaVMPoolV1WithSku(sk); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var vm *sku_lua.LuaVMV1

	if vm, args, err = sku_lua.PushTopFuncV1(lvp, args); err != nil {
		err = errors.Wrap(err)
		return err
	}

	for _, arg := range args {
		vm.Push(lua.LString(arg))
	}

	if err = vm.PCall(len(args), 0, nil); err != nil {
		err = errors.Wrap(err)
		return err
	}

	retval := vm.LState.Get(1)

	if retval.Type() != lua.LTNil {
		err = errors.ErrorWithStackf("lua error: %s", retval)
		return err
	}

	return err
}
