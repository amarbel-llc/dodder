package wasm

import (
	"context"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"github.com/tetratelabs/wazero"
)

type Runtime struct {
	inner wazero.Runtime
}

func MakeRuntime(ctx context.Context) (rt *Runtime, err error) {
	inner := wazero.NewRuntimeWithConfig(
		ctx,
		wazero.NewRuntimeConfig(),
	)

	rt = &Runtime{inner: inner}
	return rt, err
}

func (rt *Runtime) Close(ctx context.Context) error {
	return errors.Wrap(rt.inner.Close(ctx))
}
