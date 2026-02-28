package wasm

import (
	"context"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
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
				containsSku: m.ExportedFunction("contains-sku"),
				cabiRealloc: m.ExportedFunction("cabi_realloc"),
				resetFn:     m.ExportedFunction("reset"),
			}

			if mod.containsSku == nil {
				panic("WASM module missing export: contains-sku")
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
