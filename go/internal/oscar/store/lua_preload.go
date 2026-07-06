package store

import (
	"strings"

	"github.com/amarbel-llc/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/tag_blobs"
	"code.linenisgreat.com/dodder/go/lib/alfa/lua"
)

// makeHookApply composes the standard self-apply (registers Selbst + the date
// helpers) with a preload pass that registers self's blob-reference lua modules
// by name, so a hook script's `require("<basename>")` resolves them. Used by
// MakeLuaVMPoolV1 (the commit-hook VM), whose self is the committing object's
// TYPE object carrying the blob references.
func (store *Store) makeHookApply(
	self *sku.Transacted,
) interfaces.FuncIter[*lua.VM] {
	selfApply := tag_blobs.MakeLuaSelfApplyV1(self)
	return func(vm *lua.VM) (err error) {
		if err = selfApply(vm); err != nil {
			return err
		}
		return store.preloadBlobReferenceModules(vm, self)
	}
}

// preloadBlobReferenceModules registers every `.lua` blob reference on self
// (the type object) as a lua module named by its alias basename, so a hook
// script can `require("<basename>")` it. Resolution reads the referenced blob
// from the store on demand (package.preload), keeping the object-require
// searcher untouched -- the two together make the hook VM a graph-complete
// require space (FDR-0000).
func (store *Store) preloadBlobReferenceModules(
	vm *lua.VM,
	self *sku.Transacted,
) (err error) {
	if self == nil {
		return err
	}

	metadata := self.GetMetadata()

	for digest := range metadata.AllBlobReferences() {
		alias := metadata.GetBlobReferenceAlias(digest)

		name := luaModuleNameFromAlias(alias)
		if name == "" {
			continue
		}

		var stable markl.Id
		stable.ResetWithMarklId(digest) // stable copy; the Seq may reuse digest

		vm.PreloadModule(name, store.makeBlobModuleLoader(stable))
	}

	return err
}

func luaModuleNameFromAlias(alias string) string {
	if !strings.HasSuffix(alias, ".lua") {
		return ""
	}

	base := alias[strings.LastIndex(alias, "/")+1:]

	return strings.TrimSuffix(base, ".lua")
}

func (store *Store) makeBlobModuleLoader(
	digest markl.Id,
) lua.LGFunction {
	return func(ls *lua.LState) int {
		reader, err := store.GetEnvRepo().GetReadBlobStore().MakeBlobReader(digest)
		if err != nil {
			panic(errors.Wrap(err))
		}

		defer errors.DeferredCloser(&err, reader)

		var compiled *lua.FunctionProto

		if compiled, err = lua.CompileReader(reader); err != nil {
			panic(errors.Wrap(err))
		}

		ls.Push(ls.NewFunctionFromProto(compiled))

		if err = ls.PCall(0, 1, nil); err != nil {
			panic(errors.Wrap(err))
		}

		return 1
	}
}
