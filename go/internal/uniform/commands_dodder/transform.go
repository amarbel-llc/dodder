package commands_dodder

import (
	"io"
	"os"
	"slices"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/sku_lua"
	"code.linenisgreat.com/dodder/go/internal/hotel/import_plan"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/alfa/lua"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/files"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
	tap "code.linenisgreat.com/tap/go/pkgs/writer"
)

func init() {
	// FDR-0024 / RFC-0008: the list-in/list-out Lua transform over an
	// expanded inventory list. Supersedes the deleted
	// prototype-lua-transform command (Forgejo #370 item 1). See
	// docs/features/0024-inventory-list-transform-plugins.md and
	// docs/rfcs/0008-inventory-list-transform-plugin-api.md.
	utility.AddCmd("transform", &Transform{})
}

type Transform struct {
	command_components_dodder.LocalWorkingCopyWithQueryGroup

	Script         string
	ScriptDigest   string
	DryRun         bool
	SkipValidation bool
	NoNewObjects   bool
}

var (
	_ interfaces.CommandComponentWriter = (*Transform)(nil)
	_ command.CommandWithArgs           = (*Transform)(nil)
)

func (cmd *Transform) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{cmd.Query.GetArgGroup()}
}

func (cmd Transform) GetDescription() command.Description {
	return command.Description{
		Short: "run a Lua list-in/list-out transform over queried objects and commit the result",
	}
}

func (cmd *Transform) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	cmd.LocalWorkingCopyWithQueryGroup.SetFlagDefinitions(flagSet)

	flagSet.StringVar(
		&cmd.Script,
		"script",
		"",
		"path to the Lua transform script (mutually exclusive with -script-digest)",
	)

	flagSet.StringVar(
		&cmd.ScriptDigest,
		"script-digest",
		"",
		"markl id of a stored blob containing the Lua transform script (mutually exclusive with -script)",
	)

	flagSet.BoolVar(
		&cmd.DryRun,
		"dry_run",
		false,
		"build and validate the output plan and report it without committing",
	)

	flagSet.BoolVar(
		&cmd.SkipValidation,
		"skip_validation",
		false,
		"skip the fsck-style validation of the transform output (for staged, intentionally-inconsistent migration passes)",
	)

	flagSet.BoolVar(
		&cmd.NoNewObjects,
		"no_new_objects",
		false,
		"reject any output object whose object id is not present in the input list",
	)
}

func (cmd Transform) Run(req command.Request) {
	localWorkingCopy, queryGroup := cmd.MakeLocalWorkingCopyAndQueryGroup(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionDefaultSigil(
				ids.SigilLatest,
				ids.SigilHidden,
			),
			queries.BuilderOptionDefaultGenres(genres.Zettel),
		),
	)

	scriptReader, err := cmd.makeScriptReader(localWorkingCopy)
	if err != nil {
		localWorkingCopy.Cancel(err)
		return
	}

	defer errors.ContextMustClose(localWorkingCopy, scriptReader)

	list, skippedEdges, err := localWorkingCopy.MakeExpandedInventoryList(
		queryGroup,
	)
	if err != nil {
		localWorkingCopy.Cancel(errors.Wrap(err))
		return
	}

	if len(skippedEdges) > 0 {
		// A mid-migration repo (the -skip_validation use case) may hold
		// dangling references that make expansion partially fail; refusing
		// to open it would deadlock the staged-migration workflow the flag
		// exists for.
		if cmd.SkipValidation {
			localWorkingCopy.GetUI().Printf(
				"warning: edge traversal had %d failure(s); continuing due to -skip_validation",
				len(skippedEdges),
			)
		} else {
			errors.ContextCancelWithErrorf(
				localWorkingCopy,
				"edge traversal had %d failure(s): %s",
				len(skippedEdges),
				skippedEdges[0],
			)
			return
		}
	}

	var objects []*sku.Transacted
	inputIds := make(map[string]struct{})

	for object := range list.All() {
		cloned, _ := object.CloneTransacted() //repool:owned
		objects = append(objects, cloned)
		inputIds[cloned.GetObjectId().String()] = struct{}{}
	}

	localWorkingCopy.GetUI().Printf("selected %d object(s)", len(objects))

	envRepo := localWorkingCopy.GetEnvRepo()

	var binding *sku_lua.ListTransformV1

	vm, err := (&lua.VMPoolBuilder{}).WithReader(
		scriptReader,
	).WithApply(func(vm *lua.VM) error {
		binding = sku_lua.MakeListTransformV1(vm, objects)
		binding.RegisterGlobals()

		blobsTable := vm.NewTable()
		vm.SetField(blobsTable, "read", vm.NewFunction(makeLuaBlobRead(envRepo)))
		vm.SetField(blobsTable, "write", vm.NewFunction(makeLuaBlobWrite(envRepo)))
		vm.SetGlobal("blobs", blobsTable)

		return nil
	}).BuildSingleVM()
	if err != nil {
		localWorkingCopy.Cancel(errors.Wrap(err))
		return
	}

	// Single-run semantics (dodder#390): BuildSingleVM compiled the script and
	// executed the chunk exactly once during preparation, leaving the returned
	// dodder.list() handle in vm.Top. We hold this one explicitly-owned VM for
	// the whole run and Close it at the end — no pool, no repool, so the chunk
	// cannot run a second time and blobs.write cannot fire twice.
	defer vm.LState.Close()

	if binding != nil {
		defer binding.Repool()
	}

	if binding == nil || !binding.IsHandle(vm.Top) {
		errors.ContextCancelWithErrorf(
			localWorkingCopy,
			"script must return the dodder.list() handle",
		)
		return
	}

	outputs, err := binding.Objects()
	if err != nil {
		localWorkingCopy.Cancel(errors.Wrap(err))
		return
	}

	// A script reassigning one handle's Kennung onto another object's id
	// would otherwise commit two revisions onto one object (last write
	// wins) while the other object's update silently vanishes — always an
	// error, independent of -no_new_objects. Added objects (empty id,
	// allocated later) are exempt.
	seenOutputIds := make(map[string]struct{}, len(outputs))

	for _, object := range outputs {
		idString := object.GetObjectId().String()

		if idString != "" {
			if _, dupe := seenOutputIds[idString]; dupe {
				errors.ContextCancelWithErrorf(
					localWorkingCopy,
					"output contains %q more than once",
					object.GetObjectId(),
				)
				return
			}

			seenOutputIds[idString] = struct{}{}
		}

		if cmd.NoNewObjects {
			if _, ok := inputIds[idString]; !ok {
				errors.ContextCancelWithErrorf(
					localWorkingCopy,
					"-no_new_objects: output object %q is not present in the input list",
					object.GetObjectId(),
				)
				return
			}
		}
	}

	builder := import_plan.MakeLocalBuilder()
	builder.AddTransform(
		import_plan.MakeAllocateZettelIdTransform(
			localWorkingCopy.GetStore().GetZettelIdIndex(),
		),
	)

	for _, object := range outputs {
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

	plan.DefaultCommitOptions = sku.CommitOptions{
		Proto:        localWorkingCopy.GetStore().GetProtoZettel(),
		StoreOptions: sku.GetStoreOptionsUpdate(),
	}

	if plan.HasErrors {
		cmd.printPlanSummary(localWorkingCopy, plan)
		errors.ContextCancelWithErrorf(
			localWorkingCopy,
			"transform plan has errors",
		)
		return
	}

	if !cmd.SkipValidation {
		if !cmd.validate(localWorkingCopy, plan) {
			return
		}
	}

	cmd.printPlanSummary(localWorkingCopy, plan)

	if cmd.DryRun {
		localWorkingCopy.GetUI().Printf("dry run: not committed")
		return
	}

	results, err := localWorkingCopy.ExecutePlan(plan)
	if err != nil {
		localWorkingCopy.Cancel(errors.Wrap(err))
		return
	}

	localWorkingCopy.GetUI().Printf("committed %d object(s)", results.Len())
}

func (cmd Transform) makeScriptReader(
	localWorkingCopy *local_working_copy.Repo,
) (readCloser io.ReadCloser, err error) {
	switch {
	case cmd.Script != "" && cmd.ScriptDigest != "":
		err = errors.ErrorWithStackf(
			"-script and -script-digest are mutually exclusive",
		)
		return readCloser, err

	case cmd.Script != "":
		if readCloser, err = files.Open(cmd.Script); err != nil {
			err = errors.Wrapf(err, "opening -script %q", cmd.Script)
			return readCloser, err
		}

		return readCloser, err

	case cmd.ScriptDigest != "":
		var id markl.Id

		if err = id.Set(cmd.ScriptDigest); err != nil {
			err = errors.Wrapf(
				err,
				"invalid -script-digest %q",
				cmd.ScriptDigest,
			)
			return readCloser, err
		}

		if readCloser, err = localWorkingCopy.GetEnvRepo().GetReadBlobStore().MakeBlobReader(
			&id,
		); err != nil {
			err = errors.Wrapf(
				err,
				"reading -script-digest %q",
				cmd.ScriptDigest,
			)
			return readCloser, err
		}

		return readCloser, err

	default:
		err = errors.ErrorWithStackf("one of -script or -script-digest is required")
		return readCloser, err
	}
}

// validate runs the transform output through fsck's verification core
// (RFC-0008 §5). Candidate objects are pre-finalization — commit resets the
// object digest and the inventory-list flush re-signs — so the digest, sig,
// and stream-index probe checks that describe committed state are disabled;
// what remains is the blob-side safety net: blob presence for every blob
// digest and dangling blob-reference detection.
func (cmd Transform) validate(
	localWorkingCopy *local_working_copy.Repo,
	plan *import_plan.Plan,
) (ok bool) {
	var candidates []*sku.Transacted

	for i := range plan.Entries {
		entry := &plan.Entries[i]

		if !entry.Classification.IsCommittable() {
			continue
		}

		candidates = append(candidates, entry.GetObject())
	}

	tw := tap.NewWriter(os.Stdout)

	errorCount := runSeqVerification(
		localWorkingCopy,
		tw,
		quiter.MakeSeqErrorFromSeq(slices.Values(candidates)),
		seqVerificationOptions{
			SkipProbes: true,
			QuietOk:    true,
		},
	)

	if errorCount > 0 {
		errors.ContextCancelWithErrorf(
			localWorkingCopy,
			"transform output failed validation: %d error(s)",
			errorCount,
		)
		return false
	}

	return true
}

// printPlanSummary renders the plan via import's existing formatters
// (classification-count table + error tree; per-entry listing under
// -dry_run so a migration can be audited before committing).
func (cmd Transform) printPlanSummary(
	localWorkingCopy *local_working_copy.Repo,
	plan *import_plan.Plan,
) {
	if cmd.DryRun {
		plan.FormatObjects(localWorkingCopy.GetEnv().GetUIFile())
	}

	printOptions := localWorkingCopy.GetConfig().GetPrintOptions().
		WithPrintSigs(true)
	colorOptions := env_ui.FormatColorOptionsOut(localWorkingCopy, printOptions)

	boxFormatter := localWorkingCopy.StringFormatWriterSkuBoxTransacted(
		printOptions,
		colorOptions,
		string_format_writer.CliFormatTruncation66CharEllipsis,
	)

	boxFormatter.SetAbbr(plan.Abbr)
	plan.FormatSummary(localWorkingCopy.GetEnv().GetUIFile(), boxFormatter)
}

func makeLuaBlobRead(
	envRepo env_repo.Env,
) lua.LGFunction {
	return func(luaState *lua.LState) int {
		digestString := luaState.ToString(1)

		var id markl.Id

		if err := id.Set(digestString); err != nil {
			luaState.RaiseError("invalid digest %q: %s", digestString, err)
			return 0
		}

		reader, err := envRepo.GetReadBlobStore().MakeBlobReader(&id)
		if err != nil {
			luaState.RaiseError("reading blob %q: %s", digestString, err)
			return 0
		}

		defer func() {
			if closeErr := reader.Close(); closeErr != nil {
				luaState.RaiseError("closing blob reader: %s", closeErr)
			}
		}()

		body, err := io.ReadAll(reader)
		if err != nil {
			luaState.RaiseError("reading blob %q: %s", digestString, err)
			return 0
		}

		luaState.Push(lua.LString(body))

		return 1
	}
}

func makeLuaBlobWrite(
	envRepo env_repo.Env,
) lua.LGFunction {
	return func(luaState *lua.LState) int {
		body := luaState.ToString(1)

		writer, err := envRepo.GetDefaultBlobStore().MakeBlobWriter(nil)
		if err != nil {
			luaState.RaiseError("opening blob writer: %s", err)
			return 0
		}

		defer func() {
			if closeErr := writer.Close(); closeErr != nil {
				luaState.RaiseError("closing blob writer: %s", closeErr)
			}
		}()

		if _, err = writer.Write([]byte(body)); err != nil {
			luaState.RaiseError("writing blob: %s", err)
			return 0
		}

		luaState.Push(lua.LString(writer.GetMarklId().String()))

		return 1
	}
}
