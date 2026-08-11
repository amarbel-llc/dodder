package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/madder/go/pkgs/blob_store_id"
	"code.linenisgreat.com/madder/go/pkgs/blob_stores"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

func init() {
	utility.AddCmd(
		"init-from-lists",
		&InitFromLists{
			Genesis: command_components_dodder.Genesis{
				BigBang: env_repo.BigBang{
					// The union's own type objects become the repo's types, as
					// with init-from's copied types — don't genesis the default
					// !md type over them.
					ExcludeDefaultType: true,
				},
			},
		},
	)
}

// InitFromLists is the consolidation consumer of the transform pipeline
// (dodder#392): create a NEW repo — FRESH keypair, fresh uuidv7 instance
// identity, current end-state config — then apply one Lua transform to the
// UNION of N inventory-list files' object graphs and import the result. This is
// git-filter-branch into a fresh repo: the history is born already rewritten
// (tag cleanup, fork resolution, hash migration in a single pass) and re-signed
// under the newborn's key, instead of carrying legacy mess plus correction
// commits.
//
// It is distinct from init-from (copy-migration: SAME keypair, fresh uuid, a
// SINGLE source, signatures preserved). init-from answers "relocate this repo's
// identity"; init-from-lists answers "consolidate these histories into a clean
// new one." A failed run is disposable — delete the newborn and re-run.
//
// Source blobs resolve from the read-only -blob-source stores during the run,
// and every blob the committed objects reference is copied into the newborn
// before commit — so the consolidation is self-contained and survives deleting
// the (often large) -blob-source stores (dodder#392).
type InitFromLists struct {
	command_components_dodder.Genesis
	command_components_dodder.InventoryLists

	Script       string
	ScriptDigest string
	BlobSources  stringSliceFlag
}

var (
	_ interfaces.CommandComponentWriter = (*InitFromLists)(nil)
	_ command.CommandWithArgs           = (*InitFromLists)(nil)
	_ command.CommandWithResetCLIState  = (*InitFromLists)(nil)
)

func (cmd *InitFromLists) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{
			{
				Name:        "repo-id",
				Description: "location handle for the new local repository (scope via spelling: name=user, .name=cwd, //name=system)",
				Required:    true,
			},
			{
				Name:        "inventory-list-paths",
				Description: "paths to inventory list files whose object graphs are unioned",
				Required:    true,
				Variadic:    true,
			},
		},
	}}
}

func (cmd InitFromLists) GetDescription() command.Description {
	return command.Description{
		Short: "consolidate N inventory-list files into a fresh repo through a Lua transform (fresh keypair, full re-sign)",
	}
}

func (cmd *InitFromLists) SetFlagDefinitions(
	flagDefinitions interfaces.CLIFlagDefinitions,
) {
	cmd.Genesis.SetFlagDefinitions(flagDefinitions)

	flagDefinitions.StringVar(
		&cmd.Script,
		"script",
		"",
		"path to the Lua transform script (mutually exclusive with -script-digest)",
	)

	flagDefinitions.StringVar(
		&cmd.ScriptDigest,
		"script-digest",
		"",
		"markl id of a stored blob containing the Lua transform script (mutually exclusive with -script)",
	)

	flagDefinitions.Var(
		&cmd.BlobSources,
		"blob-source",
		"name of an existing madder blob store to resolve source blobs from, read-only (repeatable)",
	)
}

// ResetCLIState clears the repeatable -blob-source accumulator so a reused
// command value (the MCP bridge) does not carry one invocation's stores into
// the next.
func (cmd *InitFromLists) ResetCLIState() {
	cmd.BlobSources = nil
}

func (cmd *InitFromLists) Run(req command.Request) {
	if cmd.Script == "" && cmd.ScriptDigest == "" {
		req.Cancel(errors.BadRequestf(
			"one of -script or -script-digest is required",
		))
		return
	}

	cmd.SetLocationFromPositionalRequired(req, "new repo id")

	listPaths := req.PopArgs()
	if len(listPaths) == 0 {
		req.Cancel(errors.BadRequestf(
			"expected at least one inventory list path",
		))
		return
	}

	local := cmd.OnTheFirstDay(req)

	scriptReader, err := makeTransformScriptReader(
		local,
		cmd.Script,
		cmd.ScriptDigest,
	)
	if err != nil {
		local.Cancel(err)
		return
	}

	defer errors.ContextMustClose(local, scriptReader)

	extraReadStores, err := cmd.resolveBlobSources(local)
	if err != nil {
		local.Cancel(err)
		return
	}

	objects, err := cmd.readUnion(local, listPaths)
	if err != nil {
		local.Cancel(err)
		return
	}

	local.GetUI().Printf(
		"union of %d list(s): %d object(s)",
		len(listPaths),
		len(objects),
	)

	pipeline := transformPipeline{
		repo:         local,
		scriptReader: scriptReader,
		objects:      objects,
		// A history union carries many (id,tai) versions per id by design, and
		// fork-resolution is a deliberate same-id merge — so do NOT reject
		// duplicate object ids (dodder#392); the import builder's within-batch
		// reassign guards the genuine last-write-wins hazard instead.
		disallowDuplicateObjectIds: false,
		extraReadStores:            extraReadStores,
		// Copy every referenced source blob into the newborn before commit so
		// the consolidation is self-contained and survives deleting the
		// -blob-source stores (dodder#392).
		copyReferencedBlobsBeforeCommit: true,
		// Every object entering the newborn is re-signed under its FRESH key.
		// ExecutePlan preserves foreign signatures; CommitPlan with
		// OverwriteSignatures resets sig/pubkey/digest and re-signs
		// (FinalizeAndSignOverwrite) under the newborn's genesis key.
		commit: makeReSigningCommit(local),
	}

	if err := pipeline.run(); err != nil {
		local.Cancel(err)
		return
	}
}

func (cmd InitFromLists) resolveBlobSources(
	local *local_working_copy.Repo,
) (stores []blob_stores.BlobStoreInitialized, err error) {
	for _, idString := range cmd.BlobSources {
		var id blob_store_id.Id

		if err = id.Set(idString); err != nil {
			err = errors.Wrapf(err, "invalid -blob-source %q", idString)
			return stores, err
		}

		stores = append(
			stores,
			local.GetEnvRepo().GetEnvBlobStore().GetBlobStore(id),
		)
	}

	return stores, err
}

// readUnion reads every inventory-list file into one object list, collapsing
// exact (id,tai,digest) duplicates across the lists (see
// inventoryListUnionDeduper). The merged list is what the transform script sees.
func (cmd InitFromLists) readUnion(
	local *local_working_copy.Repo,
	paths []string,
) (objects []*sku.Transacted, err error) {
	closet := local.GetInventoryListCoderCloset()
	deduper := makeInventoryListUnionDeduper()

	for _, path := range paths {
		seq := cmd.MakeSeqFromPath(local, closet, path, nil)

		for object, iterErr := range seq {
			if iterErr != nil {
				err = errors.Wrapf(iterErr, "reading %s", path)
				return objects, err
			}

			if !deduper.keep(object) {
				continue
			}

			cloned, _ := object.CloneTransacted() //repool:owned
			objects = append(objects, cloned)
		}
	}

	return objects, err
}
