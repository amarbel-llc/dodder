package commands_dodder

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/sku_lua"
	"code.linenisgreat.com/dodder/go/internal/hotel/import_plan"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/alfa/lua"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/files"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

func init() {
	// PROTOTYPE (FDR-0024 / RFC-0008): exploring the list-in/list-out Lua
	// transform mechanism end to end before finalizing the API shape. NOT
	// the final command name/flag surface -- dry-run only for now (no
	// ExecutePlan wiring, no fsck-style validation yet). See
	// docs/features/0024-inventory-list-transform-plugins.md and
	// docs/rfcs/0008-inventory-list-transform-plugin-api.md.
	utility.AddCmd("prototype-lua-transform", &PrototypeLuaTransform{})
}

type PrototypeLuaTransform struct {
	command_components_dodder.LocalWorkingCopyWithQueryGroup

	Script string
}

var (
	_ interfaces.CommandComponentWriter = (*PrototypeLuaTransform)(nil)
	_ command.CommandWithArgs           = (*PrototypeLuaTransform)(nil)
)

func (cmd *PrototypeLuaTransform) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{cmd.Query.GetArgGroup()}
}

func (cmd PrototypeLuaTransform) GetDescription() command.Description {
	return command.Description{
		Short: "PROTOTYPE: run a Lua list-in/list-out transform over queried objects (dry-run only)",
	}
}

func (cmd *PrototypeLuaTransform) SetFlagDefinitions(f interfaces.CLIFlagDefinitions) {
	cmd.LocalWorkingCopyWithQueryGroup.SetFlagDefinitions(f)

	f.StringVar(&cmd.Script, "script", "", "path to the Lua transform script")
}

func (cmd PrototypeLuaTransform) Run(req command.Request) {
	localWorkingCopy, queryGroup := cmd.MakeLocalWorkingCopyAndQueryGroup(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionDefaultSigil(
				ids.SigilHistory,
				ids.SigilHidden,
			),
			queries.BuilderOptionDefaultGenres(genres.Zettel),
		),
	)

	if cmd.Script == "" {
		errors.ContextCancelWithErrorf(localWorkingCopy, "-script is required")
		return
	}

	list, err := localWorkingCopy.MakeInventoryList(queryGroup)
	if err != nil {
		localWorkingCopy.Cancel(err)
		return
	}

	var objects []*sku.Transacted

	for object := range list.All() {
		cloned, _ := object.CloneTransacted() //repool:owned
		objects = append(objects, cloned)
	}

	localWorkingCopy.GetUI().Printf("selected %d object(s)", len(objects))

	scriptFile, err := files.Open(cmd.Script)
	if err != nil {
		localWorkingCopy.Cancel(errors.Wrapf(err, "opening -script %q", cmd.Script))
		return
	}

	defer errors.ContextMustClose(localWorkingCopy, scriptFile)

	envRepo := localWorkingCopy.GetEnvRepo()

	var (
		array     *lua.LTable
		luaTables []*sku_lua.LuaTableV1
		repoolAll func()
	)

	vmPool, err := (&lua.VMPoolBuilder{}).WithReader(
		scriptFile,
	).WithApply(func(vm *lua.VM) error {
		tablePool := sku_lua.MakeLuaTablePoolV1(vm)
		array, luaTables, repoolAll = sku_lua.ToLuaArrayV1(vm, tablePool, objects)

		dodderTable := vm.NewTable()
		vm.SetField(dodderTable, "objects", array)
		vm.SetGlobal("dodder", dodderTable)

		blobsTable := vm.NewTable()
		vm.SetField(blobsTable, "read", vm.NewFunction(makeLuaBlobRead(envRepo)))
		vm.SetField(blobsTable, "write", vm.NewFunction(makeLuaBlobWrite(envRepo)))
		vm.SetGlobal("blobs", blobsTable)

		return nil
	}).Build()
	if err != nil {
		localWorkingCopy.Cancel(errors.Wrap(err))
		return
	}

	vm, vmRepool := vmPool.GetWithRepool()
	defer vmRepool()

	if repoolAll != nil {
		defer repoolAll()
	}

	if array == nil {
		errors.ContextCancelWithErrorf(localWorkingCopy, "script produced no object array")
		return
	}

	builder := import_plan.MakeLocalBuilder()

	for i, object := range objects {
		luaTable := luaTables[i]

		if _, err := sku_lua.FromLuaTableTransformV1(object, vm.LState, luaTable); err != nil {
			localWorkingCopy.Cancel(errors.Wrapf(err, "object %d write-back", i))
			return
		}

		if err := builder.AddObject(object, 0); err != nil {
			localWorkingCopy.Cancel(errors.Wrap(err))
			return
		}
	}

	plan, err := builder.Build()
	if err != nil {
		localWorkingCopy.Cancel(errors.Wrap(err))
		return
	}

	localWorkingCopy.GetUI().Printf(
		"plan built: %d entries, has_errors=%t (dry run -- not committed)",
		len(plan.Entries),
		plan.HasErrors,
	)

	for i := range plan.Entries {
		entry := &plan.Entries[i]
		object := entry.GetObject()

		localWorkingCopy.GetUI().Printf(
			"  %s\t%s\t%s",
			entry.Classification,
			object.GetObjectId(),
			object.GetType(),
		)
	}
}

func makeLuaBlobRead(
	envRepo env_repo.Env,
) lua.LGFunction {
	return func(ls *lua.LState) int {
		digestString := ls.ToString(1)

		var id markl.Id

		if err := id.Set(digestString); err != nil {
			ls.RaiseError("invalid digest %q: %s", digestString, err)
			return 0
		}

		reader, err := envRepo.GetLocalReadBlobStore().MakeBlobReader(&id)
		if err != nil {
			ls.RaiseError("reading blob %q: %s", digestString, err)
			return 0
		}

		defer reader.Close()

		body, err := io.ReadAll(reader)
		if err != nil {
			ls.RaiseError("reading blob %q: %s", digestString, err)
			return 0
		}

		ls.Push(lua.LString(body))

		return 1
	}
}

func makeLuaBlobWrite(
	envRepo env_repo.Env,
) lua.LGFunction {
	return func(ls *lua.LState) int {
		body := ls.ToString(1)

		writer, err := envRepo.GetDefaultBlobStore().MakeBlobWriter(nil)
		if err != nil {
			ls.RaiseError("opening blob writer: %s", err)
			return 0
		}

		defer writer.Close()

		if _, err = writer.Write([]byte(body)); err != nil {
			ls.RaiseError("writing blob: %s", err)
			return 0
		}

		ls.Push(lua.LString(writer.GetMarklId().String()))

		return 1
	}
}
