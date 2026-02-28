package wasm

import (
	"context"
	"io"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"github.com/tetratelabs/wazero"
)

type ModulePoolBuilder struct {
	runtime  *Runtime
	wasmData []byte
	compiled wazero.CompiledModule
}

func MakeModulePoolBuilder(rt *Runtime) *ModulePoolBuilder {
	return &ModulePoolBuilder{runtime: rt}
}

func (b *ModulePoolBuilder) WithBytes(data []byte) *ModulePoolBuilder {
	b.wasmData = data
	return b
}

func (b *ModulePoolBuilder) WithReader(r io.Reader) *ModulePoolBuilder {
	data, err := io.ReadAll(r)
	if err != nil {
		panic(errors.Wrap(err))
	}

	b.wasmData = data
	return b
}

func (b *ModulePoolBuilder) WithCompiled(
	compiled wazero.CompiledModule,
) *ModulePoolBuilder {
	b.compiled = compiled
	return b
}

func (b *ModulePoolBuilder) Build(
	ctx context.Context,
) (mp *ModulePool, err error) {
	if b.compiled == nil && b.wasmData == nil {
		err = errors.ErrorWithStackf("no WASM data or compiled module set")
		return mp, err
	}

	if b.compiled == nil {
		if b.compiled, err = b.runtime.inner.CompileModule(
			ctx,
			b.wasmData,
		); err != nil {
			err = errors.Wrap(err)
			return mp, err
		}
	}

	mp = makeModulePool(ctx, b.runtime, b.compiled)

	// Verify the module works by borrowing and returning one instance.
	mod, repool := mp.GetWithRepool()
	defer repool()
	_ = mod

	return mp, err
}
