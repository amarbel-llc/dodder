package lua

import (
	"io"
	"strings"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
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

// BuildSingleVM constructs a single, caller-owned VM instead of a pool: it
// compiles the script once, creates one sandboxed LState, and runs the
// module preload, searcher, apply hook, and the compiled chunk exactly once
// (via PrepareVM), leaving the chunk's return value in vm.Top. The caller owns
// the returned VM and MUST Close it when done.
//
// This is the single-run path for one-shot batch callers (the transform
// command, dodder#390): the chunk executes exactly once, with no sync.Pool
// borrow/repool and therefore none of the trial-VM/GC re-execution window the
// pooled Build path has. The pool (Build) remains the right choice for the
// repeated per-object tag-filter workload it was designed for.
func (vpb *VMPoolBuilder) BuildSingleVM() (vm *VM, err error) {
	compiled := vpb.compiled

	if compiled == nil {
		if vpb.scriptReader == nil {
			err = errors.ErrorWithStackf("no script, reader, or compiled set")
			return vm, err
		}

		if compiled, err = CompileReader(vpb.scriptReader); err != nil {
			err = errors.Wrap(err)
			return vm, err
		}
	}

	vmp := &VMPool{
		Require:  vpb.proto.Require,
		Searcher: vpb.proto.Searcher,
		compiled: compiled,
	}

	vm = &VM{
		LState: lua.NewState(lua.Options{SkipOpenLibs: true}),
	}
	openSafeLibs(vm.LState)

	if err = vmp.PrepareVM(vm, vpb.apply); err != nil {
		vm.LState.Close()
		err = errors.Wrap(err)
		return nil, err
	}

	// Match Build()'s trial-VM check: the chunk must return a table. Callers
	// (the transform command) layer stricter checks on vm.Top on top of this.
	if _, err = vm.GetTopTableOrError(); err != nil {
		vm.LState.Close()
		err = errors.Wrap(err)
		return nil, err
	}

	return vm, err
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
