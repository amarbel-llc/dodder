package lua

import (
	"io"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/pool"
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
		// der/dodder/zit are three aliases for the same require entrypoint (zit
		// is a legacy name, eventually to be removed). Preload each with one
		// shared module loader and expose each as a global backed by one shared
		// table, rather than three byte-identical closures (dodder#391).
		loadRequireModule := func(s *lua.LState) int {
			mod := s.SetFuncs(s.NewTable(), map[string]lua.LGFunction{
				"require": sp.Require,
			})

			s.Push(mod)

			return 1
		}

		table, _ := vm.PoolPtr.GetWithRepool() //repool:owned
		vm.SetField(table, "require", vm.NewFunction(sp.Require))

		for _, name := range []string{"der", "dodder", "zit"} {
			vm.PreloadModule(name, loadRequireModule)
			vm.SetGlobal(name, table)
		}
	}

	if sp.Searcher != nil {
		// Insert the custom der/dodder/zit searcher at loaders[1] (ahead of the
		// preload searcher). This runs AFTER openSafeLibs replaced the
		// filesystem searcher at loaders[2], so the insert shifts every entry up
		// one index — which is exactly why openSafeLibs blocks the filesystem
		// searcher at construction only and applySandboxRestrictions must NOT
		// re-run that block on repool (it would then clobber loaders[2], now the
		// preload searcher). See openSafeLibs in stdlib.go.
		packageTable := vm.GetGlobal("package").(*LTable)
		loaderTable := vm.GetField(packageTable, "loaders").(*LTable)
		loaderTable.Insert(1, vm.NewFunction(sp.Searcher))
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
				LState: lua.NewState(lua.Options{SkipOpenLibs: true}),
			}
			openSafeLibs(vm.LState)

			if err := sp.PrepareVM(vm, apply); err != nil {
				panic(errors.Wrap(err))
			}

			return vm
		},
		func(vm *VM) {
			vm.SetTop(0)
			// The pool (sync.Pool) hands the same LState back on the next
			// borrow, so re-arm the sandbox: without this, a script that
			// overwrote dofile or the io/os proxy would leak that mutation
			// into the next script sharing this VM slot. Cheap (a handful of
			// SetGlobal calls) and unconditional — sandbox integrity is an
			// invariant, not a per-borrow policy knob.
			applySandboxRestrictions(vm.LState)
		},
	)

	return err
}
