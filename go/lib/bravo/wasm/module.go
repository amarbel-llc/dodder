package wasm

import (
	"context"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"github.com/tetratelabs/wazero/api"
)

type Module struct {
	mod         api.Module
	memory      api.Memory
	containsSku api.Function
	cabiRealloc api.Function
	resetFn     api.Function
}

func (m *Module) CallContainsSku(
	ctx context.Context,
	recordPtr uint32,
) (bool, error) {
	results, err := m.containsSku.Call(ctx, uint64(recordPtr))
	if err != nil {
		return false, errors.Wrap(err)
	}

	return results[0] != 0, nil
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
