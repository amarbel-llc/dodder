package commands_dodder

import (
	"io"
	"os"
	"slices"

	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/blob_transfers"
	"code.linenisgreat.com/dodder/go/internal/golf/sku_lua"
	"code.linenisgreat.com/dodder/go/internal/hotel/import_plan"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/alfa/lua"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter"
	"code.linenisgreat.com/madder/go/pkgs/blob_stores"
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/files"
	tap "code.linenisgreat.com/tap/go/pkgs/writer"
)

// makeTransformScriptReader opens the transform script from either a local file
// (-script) or a stored blob addressed by markl id (-script-digest, via the
// multi-store read fallback). The two are mutually exclusive and one is
// required. Shared by every transform-pipeline consumer.
func makeTransformScriptReader(
	repo *local_working_copy.Repo,
	scriptPath string,
	scriptDigest string,
) (readCloser io.ReadCloser, err error) {
	switch {
	case scriptPath != "" && scriptDigest != "":
		err = errors.ErrorWithStackf(
			"-script and -script-digest are mutually exclusive",
		)
		return readCloser, err

	case scriptPath != "":
		if readCloser, err = files.Open(scriptPath); err != nil {
			err = errors.Wrapf(err, "opening -script %q", scriptPath)
			return readCloser, err
		}

		return readCloser, err

	case scriptDigest != "":
		var id markl.Id

		if err = id.Set(scriptDigest); err != nil {
			err = errors.Wrapf(err, "invalid -script-digest %q", scriptDigest)
			return readCloser, err
		}

		if readCloser, err = repo.GetEnvRepo().GetReadBlobStore().MakeBlobReader(
			&id,
		); err != nil {
			err = errors.Wrapf(err, "reading -script-digest %q", scriptDigest)
			return readCloser, err
		}

		return readCloser, err

	default:
		err = errors.ErrorWithStackf(
			"one of -script or -script-digest is required",
		)
		return readCloser, err
	}
}

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

	// disallowDuplicateObjectIds rejects a script whose output names the same
	// object id more than once. `transform` sets it (its query source yields
	// one latest version per id, so two same-id outputs mean the script merged
	// two objects onto one id, silently losing one under last-write-wins). The
	// inventory-list consumers (init-from-lists, clone -script) leave it OFF:
	// a history union carries many (id,tai) versions per id BY DESIGN, and
	// ruled fork-resolution is a deliberate same-id merge (dodder#392) — the
	// import builder's within-batch (id,tai) reassign guards the genuine
	// last-write-wins hazard instead. Do not "fix" this asymmetry.
	disallowDuplicateObjectIds bool

	// extraReadStores are read-only blob stores consulted ahead of the repo's
	// own read view (and, under -dry_run, ahead of the staging store) by the
	// blobs.read FFI and dry-run validation. init-from-lists passes its
	// -blob-source stores here so the script can read source blobs and a dry
	// run can validate against them. A real run does NOT validate against these
	// — it copies referenced blobs into the target first (see
	// copyReferencedBlobsBeforeCommit) and validates the target. Empty for
	// `transform`.
	extraReadStores []blob_stores.BlobStoreInitialized

	// copyReferencedBlobsBeforeCommit copies every blob referenced by the
	// committable output (each object's own Blob plus its field-level
	// file<@digest references), if missing from the write store, out of the
	// read view before the real commit. The inventory-list consumers set it so
	// the target is SELF-CONTAINED: source blobs resolve via -blob-source during
	// the run but are duplicated into the newborn, so the consolidation survives
	// deleting the (large) legacy sources — the program's terminal step
	// (dodder#392). Skipped under -dry_run (nothing commits; the overlay serves
	// validation) and a no-op for `transform` (its blobs are already local).
	copyReferencedBlobsBeforeCommit bool

	// commit commits the built plan into the target repo and returns the number
	// of objects committed. `transform` uses ExecutePlan (its objects are
	// locally-authored and sealed under this repo's key at the working-list
	// flush). The inventory-list consumers use remote_transfer.CommitPlan with
	// OverwriteSignatures so foreign objects are re-signed under the newborn's
	// key — ExecutePlan does NOT re-sign (dodder#392).
	commit func(*import_plan.Plan) (int, error)
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

	// -blob-source read-only stores (init-from-lists consolidation) are
	// consulted ahead of the repo's own read view by the blobs.read FFI, so the
	// script can read source blobs. A real run's validation reads the target
	// instead (the referenced blobs are copied in before commit — see below);
	// only a dry run validates against this overlay (set in the dry-run branch).
	if len(p.extraReadStores) > 0 {
		readStore = envRepo.MakeReadBlobStoreWithOverlay(p.extraReadStores...)
	}

	if p.dryRun {
		stagingStore, dir, err := envRepo.MakeDiscardableStagingBlobStore()
		if err != nil {
			return errors.Wrap(err)
		}

		// Blobs staged by blobs.write live only in the staging store, so both
		// the blobs.read FFI and the output validation must read through an
		// overlay that consults staging (then any -blob-source stores) before
		// the real read view — otherwise an object whose Blob was rewritten to
		// a staged digest would fail validation, defeating a dry run of the
		// very migrations this exists for.
		overlays := append(
			[]blob_stores.BlobStoreInitialized{stagingStore},
			p.extraReadStores...,
		)
		overlay := envRepo.MakeReadBlobStoreWithOverlay(overlays...)

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

		if p.disallowDuplicateObjectIds && idString != "" {
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

	// Self-containment (dodder#392): before committing foreign objects, copy
	// every referenced blob missing from the write store out of the read view
	// (which includes -blob-source) so the target survives deleting the sources.
	// This makes store.Commit resolve every blob natively from the target and
	// lets validation read the target rather than the overlay. Skipped under
	// -dry_run (nothing commits; the overlay serves validation) and a no-op for
	// `transform` (its blobs are already local).
	if p.copyReferencedBlobsBeforeCommit && !p.dryRun {
		if err := copyReferencedBlobsIntoWriteStore(
			envRepo,
			plan,
			readStore,
			p.skipValidation,
		); err != nil {
			return errors.Wrap(err)
		}
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

	committed, err := p.commit(plan)
	if err != nil {
		return errors.Wrap(err)
	}

	repo.GetUI().Printf("committed %d object(s)", committed)

	return nil
}

// copyReferencedBlobsIntoWriteStore streams every blob the committable output
// references — each object's own Blob plus its field-level file<@digest
// references — from src into the target repo's default (write) store, if not
// already present. This makes a consolidation self-contained: the referenced
// blobs are duplicated into the newborn so it survives deleting the -blob-source
// stores (dodder#392). Copies are content-addressed and skipped when present,
// so re-runs are cheap. When tolerateMissing is set (-skip_validation, staged
// intermediate passes) a blob absent from every source is skipped rather than
// erroring.
func copyReferencedBlobsIntoWriteStore(
	envRepo env_repo.Env,
	plan *import_plan.Plan,
	src mad_domain_interfaces.BlobStore,
	tolerateMissing bool,
) error {
	blobImporter := blob_transfers.MakeBlobImporter(
		envRepo.GetEnvBlobStore(),
		src,
		blob_stores.MakeBlobStoreMap(envRepo.GetDefaultBlobStore()),
	)

	copyOne := func(
		blobId mad_domain_interfaces.MarklId,
		object *sku.Transacted,
	) error {
		if err := blobImporter.ImportBlobIfNecessary(blobId, object); err != nil {
			if tolerateMissing {
				return nil
			}

			return errors.Wrapf(err, "copying referenced blob %s", blobId)
		}

		return nil
	}

	for i := range plan.Entries {
		entry := &plan.Entries[i]

		if !entry.Classification.IsCommittable() {
			continue
		}

		object := entry.GetObject()
		metadata := object.GetMetadata()

		if blobDigest := metadata.GetBlobDigest(); !blobDigest.IsNull() {
			if err := copyOne(blobDigest, object); err != nil {
				return err
			}
		}

		for refDigest := range metadata.AllBlobReferences() {
			if err := copyOne(refDigest, object); err != nil {
				return err
			}
		}
	}

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
