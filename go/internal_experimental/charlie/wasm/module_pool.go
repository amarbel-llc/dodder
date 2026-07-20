package wasm

import (
	"context"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/pool"
	"github.com/tetratelabs/wazero"
)

type ModulePool struct {
	interfaces.PoolPtr[Module, *Module]
	compiled wazero.CompiledModule
	runtime  *Runtime
	ctx      context.Context
}

func makeModulePool(
	ctx context.Context,
	rt *Runtime,
	compiled wazero.CompiledModule,
) *ModulePool {
	mp := &ModulePool{
		compiled: compiled,
		runtime:  rt,
		ctx:      ctx,
	}

	mp.PoolPtr = pool.Make(
		func() (mod *Module) {
			m, err := rt.inner.InstantiateModule(
				ctx,
				compiled,
				wazero.NewModuleConfig().WithName(""),
			)
			if err != nil {
				panic(errors.Wrap(err))
			}

			mod = &Module{
				mod:         m,
				memory:      m.Memory(),
				cabiRealloc: m.ExportedFunction("cabi_realloc"),
				resetFn:     m.ExportedFunction("reset"),
			}

			if mod.cabiRealloc == nil {
				panic("WASM module missing export: cabi_realloc")
			}

			return mod
		},
		func(mod *Module) {
			if mod.resetFn != nil {
				if err := mod.CallReset(ctx); err != nil {
					panic(errors.Wrap(err))
				}
			}
		},
	)

	return mp
}
