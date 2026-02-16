package lua

import (
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	lua "github.com/yuin/gopher-lua"
)

type VMPoolBuilder struct {
	proto        VMPool
	scriptReader io.Reader
	compiled     *lua.FunctionProto
	apply        interfaces.FuncIter[*VM]
}

func (vpb *VMPoolBuilder) Clone() *VMPoolBuilder {
	clone := *vpb
	// TODO support cloning of vpb.compiled
	return &clone
}

func (vpb *VMPoolBuilder) WithRequire(v LGFunction) *VMPoolBuilder {
	vpb.proto.Require = v
	return vpb
}

func (vpb *VMPoolBuilder) WithSearcher(v LGFunction) *VMPoolBuilder {
	vpb.proto.Searcher = v
	return vpb
}

func (sp *VMPoolBuilder) WithScript(
	script string,
) *VMPoolBuilder {
	sp.scriptReader = strings.NewReader(script)
	return sp
}

func (sp *VMPoolBuilder) WithReader(
	r io.Reader,
) *VMPoolBuilder {
	sp.scriptReader = r
	return sp
}

func (sp *VMPoolBuilder) WithCompiled(
	compiled *FunctionProto,
) *VMPoolBuilder {
	sp.compiled = compiled
	return sp
}

func (sp *VMPoolBuilder) WithApply(
	apply interfaces.FuncIter[*VM],
) *VMPoolBuilder {
	sp.apply = apply
	return sp
}

func (vpb *VMPoolBuilder) Build() (vmp *VMPool, err error) {
	vmp = &VMPool{
		Require:  vpb.proto.Require,
		Searcher: vpb.proto.Searcher,
	}

	if vpb.scriptReader == nil && vpb.compiled == nil {
		err = errors.ErrorWithStackf("no script, reader, or compiled set")
		return vmp, err
	}

	if vpb.compiled != nil {
		if err = vmp.SetCompiled(vpb.compiled, vpb.apply); err != nil {
			err = errors.Wrap(err)
			return vmp, err
		}
	} else if vpb.scriptReader != nil {
		if err = vmp.SetReader(vpb.scriptReader, vpb.apply); err != nil {
			err = errors.Wrap(err)
			return vmp, err
		}
	}

	// try initializing a lua vm to make sure there are no errors
	vm, repool := vmp.GetWithRepool()
	defer repool()

	if _, err = vm.GetTopTableOrError(); err != nil {
		err = errors.Wrap(err)
		return vmp, err
	}

	return vmp, err
}

func MakeVMPoolWithSearcher(
	script string,
	searcher LGFunction,
	apply interfaces.FuncIter[*VM],
) (ml *VMPool, err error) {
	b := (&VMPoolBuilder{}).WithSearcher(searcher).WithScript(script).WithApply(apply)

	if ml, err = b.Build(); err != nil {
		err = errors.Wrap(err)
		return ml, err
	}

	return ml, err
}

func MakeVMPoolWithRequire(
	script string,
	require LGFunction,
	apply interfaces.FuncIter[*VM],
) (ml *VMPool, err error) {
	b := (&VMPoolBuilder{}).WithRequire(require).WithScript(script).WithApply(apply)

	if ml, err = b.Build(); err != nil {
		err = errors.Wrap(err)
		return ml, err
	}

	return ml, err
}
