package wasm

import (
	"context"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"github.com/tetratelabs/wazero/api"
)

type Module struct {
	mod         api.Module
	memory      api.Memory
	cabiRealloc api.Function
	resetFn     api.Function
}

func (m *Module) Inner() api.Module {
	return m.mod
}

func (m *Module) CallCabiRealloc(
	ctx context.Context,
	oldPtr, oldSize, align, newSize uint32,
) (uint32, error) {
	results, err := m.cabiRealloc.Call(
		ctx,
		uint64(oldPtr), uint64(oldSize),
		uint64(align), uint64(newSize),
	)
	if err != nil {
		return 0, errors.Wrap(err)
	}

	return uint32(results[0]), nil
}

func (m *Module) CallReset(ctx context.Context) error {
	_, err := m.resetFn.Call(ctx)
	return errors.Wrap(err)
}

func (m *Module) Memory() api.Memory {
	return m.memory
}

func (m *Module) WriteBytes(offset uint32, data []byte) bool {
	return m.memory.Write(offset, data)
}

func (m *Module) ReadBytes(offset, size uint32) ([]byte, bool) {
	return m.memory.Read(offset, size)
}
