package lua

import (
	"io"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	lua "github.com/yuin/gopher-lua"
)

type VMPool struct {
	interfaces.PoolPtr[VM, *VM]
	Require  LGFunction
	Searcher LGFunction
	compiled *lua.FunctionProto
}

func (sp *VMPool) PrepareVM(
	vm *VM,
	apply interfaces.FuncIter[*VM],
) (err error) {
	vm.PoolPtr = pool.Make(
		func() (t *lua.LTable) {
			t = vm.NewTable()
			return t
		},
		func(t *lua.LTable) {
			ClearTable(vm.LState, t)
		},
	)

	if sp.Require != nil {
		vm.PreloadModule("der", func(s *lua.LState) int {
			// register functions to the table
			mod := s.SetFuncs(s.NewTable(), map[string]lua.LGFunction{
				"require": sp.Require,
			})

			s.Push(mod)

			return 1
		})

		vm.PreloadModule("dodder", func(s *lua.LState) int {
			// register functions to the table
			mod := s.SetFuncs(s.NewTable(), map[string]lua.LGFunction{
				"require": sp.Require,
			})

			s.Push(mod)

			return 1
		})

		// TODO eventually remove
		vm.PreloadModule("zit", func(s *lua.LState) int {
			// register functions to the table
			mod := s.SetFuncs(s.NewTable(), map[string]lua.LGFunction{
				"require": sp.Require,
			})

			s.Push(mod)

			return 1
		})

		table, _ := vm.PoolPtr.GetWithRepool()
		vm.SetField(table, "require", vm.NewFunction(sp.Require))
		vm.SetGlobal("der", table)
		vm.SetGlobal("dodder", table)
		vm.SetGlobal("zit", table)
	}

	if sp.Searcher != nil {
		packageTable := vm.GetGlobal("package").(*LTable)

		if true { // lua <= 5.1
			loaderTable := vm.GetField(packageTable, "loaders").(*LTable)
			loaderTable.Insert(1, vm.NewFunction(sp.Searcher))
		} else {
			searcherTable, _ := vm.PoolPtr.GetWithRepool()
			packageTable.Insert(1, searcherTable)
			searcherTable.Insert(1, vm.NewFunction(sp.Searcher))
		}
	}

	if apply != nil {
		if err = apply(vm); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	lfunc := vm.NewFunctionFromProto(sp.compiled)
	vm.Push(lfunc)

	if err = vm.PCall(0, 1, nil); err != nil {
		err = errors.Wrap(err)
		return err
	}

	vm.Top = vm.LState.Get(1)
	vm.Pop(1)

	return err
}

func (sp *VMPool) SetReader(
	reader io.Reader,
	apply interfaces.FuncIter[*VM],
) (err error) {
	var compiled *FunctionProto

	if compiled, err = CompileReader(reader); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = sp.SetCompiled(compiled, apply); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (sp *VMPool) SetCompiled(
	compiled *FunctionProto,
	apply interfaces.FuncIter[*VM],
) (err error) {
	sp.compiled = compiled

	sp.PoolPtr = pool.Make(
		func() (vm *VM) {
			vm = &VM{
				LState: lua.NewState(),
			}

			if err := sp.PrepareVM(vm, apply); err != nil {
				panic(errors.Wrap(err))
			}

			return vm
		},
		func(vm *VM) {
			vm.SetTop(0)
		},
	)

	return err
}
