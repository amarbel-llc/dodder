package lua

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	lua "github.com/yuin/gopher-lua"
)

type VM struct {
	*lua.LState
	Top lua.LValue
	interfaces.PoolPtr[LTable, *LTable]
}

func (vm *VM) GetTopFunctionOrFunctionNamedError(
	args []string,
) (t *LFunction, argsOut []string, err error) {
	funcName := ""
	argsOut = args

	if len(args) > 0 {
		funcName = args[0]
	}

	if vm.Top.Type() == LTTable {
		if funcName == "" {
			err = errors.ErrorWithStackf("needs function name because top is table")
			return t, argsOut, err
		}

		tt := vm.Top.(*LTable)

		tv := vm.GetField(tt, funcName)

		if tv.Type() != LTFunction {
			err = errors.ErrorWithStackf("expected %v but got %v", LTFunction, tv.Type())
			return t, argsOut, err
		}

		argsOut = argsOut[1:]

		t = tv.(*LFunction)
	} else if vm.Top.Type() == LTFunction {
		t = vm.Top.(*LFunction)
	} else {
		err = errors.ErrorWithStackf(
			"expected table or function but got: %q",
			vm.Top.Type(),
		)

		return t, argsOut, err
	}

	return t, argsOut, err
}

func (vm *VM) GetTopTableOrError() (t *LTable, err error) {
	if vm.Top.Type() != LTTable {
		err = errors.ErrorWithStackf("expected %v but got %v", LTTable, vm.Top.Type())
		return t, err
	}

	t = vm.Top.(*LTable)

	return t, err
}

func (vm *VM) GetTopFunctionOrError() (t *LFunction, err error) {
	if vm.Top.Type() != LTFunction {
		err = errors.ErrorWithStackf("expected %v but got %v", LTFunction, vm.Top.Type())
		return t, err
	}

	t = vm.Top.(*LFunction)

	return t, err
}
