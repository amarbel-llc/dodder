package commands_dodder

import (
	"io"
	"os"
	"slices"

	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/sku_lua"
	"code.linenisgreat.com/dodder/go/internal/hotel/import_plan"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/alfa/lua"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter"
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	tap "code.linenisgreat.com/tap/go/pkgs/writer"
)

// transformPipeline is the source-agnostic core of the FDR-0024/RFC-0008
// transform mechanism, shared by its consumers (dodder#392): `transform`
// (query source), `init` (inventory-list union source), and `clone -script`
// (pull-stream source). Given a source list of objects, a script, and a target
// repo, it runs the sandboxed single-run Lua VM (#389/#390) over the objects,
// checks the output, builds an import plan, validates it, and either reports a
// dry run (blobs.write contained in a discardable staging store, #390) or
// commits into the target repo.
//
// The consumers differ only in how they produce `objects` and which repo they
// target; everything from the VM onward lives here. Errors are returned rather
// than cancelled in place so each consumer owns its context; validation and
// plan summaries still write to the target repo's UI.
type transformPipeline struct {
	repo           *local_working_copy.Repo
	scriptReader   io.Reader
	objects        []*sku.Transacted
	dryRun         bool
	skipValidation bool
	noNewObjects   bool
}

func (p transformPipeline) run() error {
	repo := p.repo
	envRepo := repo.GetEnvRepo()

	inputIds := make(map[string]struct{}, len(p.objects))
	for _, object := range p.objects {
		inputIds[object.GetObjectId().String()] = struct{}{}
	}

	// Wire the blob FFI. A real run writes to the default store and reads
	// through the multi-store view. A dry run (dodder#390) must not touch the
	// real store, so blobs.write goes to a discardable staging store and
	// blobs.read overlays that staging store over the real read view
	// (preserving read-your-writes within the run); staged digests are recorded
	// so the summary can surface what landed and where.
	writeStore := mad_domain_interfaces.BlobStore(envRepo.GetDefaultBlobStore())
	readStore := mad_domain_interfaces.BlobStore(envRepo.GetReadBlobStore())

	var stagingDir string
	var stagedDigests []string
	var onStaged func(digest string)
	var validationReadStore mad_domain_interfaces.BlobStore

	if p.dryRun {
		stagingStore, dir, err := envRepo.MakeDiscardableStagingBlobStore()
		if err != nil {
			return errors.Wrap(err)
		}

		// Blobs staged by blobs.write live only in the staging store, so both
		// the blobs.read FFI and the output validation must read through an
		// overlay that consults staging before the real read view — otherwise
		// an object whose Blob was rewritten to a staged digest would fail
		// validation, defeating a dry run of the very migrations this exists
		// for.
		overlay := envRepo.MakeReadBlobStoreWithOverlay(stagingStore)

		stagingDir = dir
		writeStore = stagingStore
		readStore = overlay
		validationReadStore = overlay
		onStaged = func(digest string) { stagedDigests = append(stagedDigests, digest) }
	}

	var binding *sku_lua.ListTransformV1

	vm, err := (&lua.VMPoolBuilder{}).WithReader(
		p.scriptReader,
	).WithApply(func(vm *lua.VM) error {
		binding = sku_lua.MakeListTransformV1(vm, p.objects)
		binding.RegisterGlobals()

		blobsTable := vm.NewTable()
		vm.SetField(blobsTable, "read", vm.NewFunction(makeLuaBlobRead(readStore)))
		vm.SetField(blobsTable, "write", vm.NewFunction(makeLuaBlobWrite(writeStore, onStaged)))
		vm.SetGlobal("blobs", blobsTable)

		return nil
	}).BuildSingleVM()
	if err != nil {
		return errors.Wrap(err)
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
		return errors.ErrorWithStackf("script must return the dodder.list() handle")
	}

	outputs, err := binding.Objects()
	if err != nil {
		return errors.Wrap(err)
	}

	// A script reassigning one handle's Kennung onto another object's id would
	// otherwise commit two revisions onto one object (last write wins) while
	// the other object's update silently vanishes — always an error,
	// independent of -no_new_objects. Added objects (empty id, allocated later)
	// are exempt.
	seenOutputIds := make(map[string]struct{}, len(outputs))

	for _, object := range outputs {
		idString := object.GetObjectId().String()

		if idString != "" {
			if _, dupe := seenOutputIds[idString]; dupe {
				return errors.ErrorWithStackf(
					"output contains %q more than once",
					object.GetObjectId(),
				)
			}

			seenOutputIds[idString] = struct{}{}
		}

		if p.noNewObjects {
			if _, ok := inputIds[idString]; !ok {
				return errors.ErrorWithStackf(
					"-no_new_objects: output object %q is not present in the input list",
					object.GetObjectId(),
				)
			}
		}
	}

	builder := import_plan.MakeLocalBuilder()
	builder.AddTransform(
		import_plan.MakeAllocateZettelIdTransform(
			repo.GetStore().GetZettelIdIndex(),
		),
	)

	for _, object := range outputs {
		if err := builder.AddObject(object, 0); err != nil {
			return errors.Wrap(err)
		}
	}

	plan, err := builder.Build()
	if err != nil {
		return errors.Wrap(err)
	}

	plan.DefaultCommitOptions = sku.CommitOptions{
		Proto:        repo.GetStore().GetProtoZettel(),
		StoreOptions: sku.GetStoreOptionsUpdate(),
	}

	if plan.HasErrors {
		p.printPlanSummary(plan)
		return errors.ErrorWithStackf("transform plan has errors")
	}

	if !p.skipValidation {
		if err := p.validate(plan, validationReadStore); err != nil {
			return err
		}
	}

	p.printPlanSummary(plan)

	if p.dryRun {
		p.reportStagedBlobs(stagingDir, stagedDigests)
		repo.GetUI().Printf("dry run: not committed")
		return nil
	}

	results, err := repo.ExecutePlan(plan)
	if err != nil {
		return errors.Wrap(err)
	}

	repo.GetUI().Printf("committed %d object(s)", results.Len())

	return nil
}

// validate runs the transform output through fsck's verification core
// (RFC-0008 §5). Candidate objects are pre-finalization — commit resets the
// object digest and the inventory-list flush re-signs — so the digest, sig,
// and stream-index probe checks that describe committed state are disabled;
// what remains is the blob-side safety net: blob presence for every blob
// digest and dangling blob-reference detection. Returns nil when the output
// verifies; a non-nil error (with the TAP not-ok lines already written)
// otherwise.
func (p transformPipeline) validate(
	plan *import_plan.Plan,
	readBlobStore mad_domain_interfaces.BlobStore,
) error {
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
		p.repo,
		tw,
		quiter.MakeSeqErrorFromSeq(slices.Values(candidates)),
		seqVerificationOptions{
			SkipProbes:    true,
			QuietOk:       true,
			ReadBlobStore: readBlobStore,
		},
	)

	if errorCount > 0 {
		return errors.ErrorWithStackf(
			"transform output failed validation: %d error(s)",
			errorCount,
		)
	}

	return nil
}

// printPlanSummary renders the plan via import's existing formatters
// (classification-count table + error tree; per-entry listing under -dry_run
// so a migration can be audited before committing).
func (p transformPipeline) printPlanSummary(plan *import_plan.Plan) {
	repo := p.repo

	if p.dryRun {
		plan.FormatObjects(repo.GetEnv().GetUIFile())
	}

	printOptions := repo.GetConfig().GetPrintOptions().
		WithPrintSigs(true)
	colorOptions := env_ui.FormatColorOptionsOut(repo, printOptions)

	boxFormatter := repo.StringFormatWriterSkuBoxTransacted(
		printOptions,
		colorOptions,
		string_format_writer.CliFormatTruncation66CharEllipsis,
	)

	boxFormatter.SetAbbr(plan.Abbr)
	plan.FormatSummary(repo.GetEnv().GetUIFile(), boxFormatter)
}

// reportStagedBlobs surfaces what a dry run's blobs.write calls staged and
// where, so the results can be inspected before a real run. The staging
// directory is always safe to delete. When nothing was staged the empty run
// directory is removed so repeated dry runs don't litter the cache.
func (p transformPipeline) reportStagedBlobs(
	stagingDir string,
	stagedDigests []string,
) {
	if stagingDir == "" {
		return
	}

	if len(stagedDigests) == 0 {
		// nothing landed; best-effort so no empty run dir is left behind
		_ = os.Remove(stagingDir)
		return
	}

	p.repo.GetUI().Printf(
		"dry run: staged %d blob(s) under %s (safe to delete)",
		len(stagedDigests),
		stagingDir,
	)

	for _, digest := range stagedDigests {
		p.repo.GetUI().Printf("  staged blob %s", digest)
	}
}

// makeLuaBlobRead reads a blob by digest from readStore. A real run passes the
// multi-store read view; a dry run passes an overlay that consults the staging
// store before that view, so blobs.write output earlier in the same run is
// readable back (the overlay handles the fallback internally).
func makeLuaBlobRead(
	readStore mad_domain_interfaces.BlobStore,
) lua.LGFunction {
	return func(luaState *lua.LState) int {
		digestString := luaState.ToString(1)

		var id markl.Id

		if err := id.Set(digestString); err != nil {
			luaState.RaiseError("invalid digest %q: %s", digestString, err)
			return 0
		}

		reader, err := readStore.MakeBlobReader(&id)
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

// makeLuaBlobWrite writes a blob to writeStore and returns its digest. On a real
// run writeStore is the default store and onStaged is nil. Under -dry_run
// writeStore is a discardable staging store (never the repo's real store) and
// onStaged records each digest so the dry-run summary can surface what was
// staged.
func makeLuaBlobWrite(
	writeStore mad_domain_interfaces.BlobStore,
	onStaged func(digest string),
) lua.LGFunction {
	return func(luaState *lua.LState) int {
		body := luaState.ToString(1)

		writer, err := writeStore.MakeBlobWriter(nil)
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

		digest := writer.GetMarklId().String()

		if onStaged != nil {
			onStaged(digest)
		}

		luaState.Push(lua.LString(digest))

		return 1
	}
}
