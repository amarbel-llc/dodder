package repo_actions

import (
	"fmt"
	"sync"

	"code.linenisgreat.com/dodder/go/internal/bravo/checked_out_state"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/import_plan"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/kilo/orgie"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

func MakeOrganizeOptionsWithOrganizeMetadata(
	repo *local_working_copy.Repo,
	organizeFlags orgie.Flags,
	metadata orgie.Metadata,
) orgie.Options {
	options := organizeFlags.GetOptions(
		repo.GetConfig().GetPrintOptions(),
		nil,
		repo.SkuFormatBoxCheckedOutNoColor(),
		repo.GetStore().GetAbbrStore().GetAbbr(),
		sku.ObjectFactory{},
	)

	options.Metadata = metadata

	return options
}

func MakeOrganizeOptionsWithQueryGroup(
	repo *local_working_copy.Repo,
	organizeFlags orgie.Flags,
	qg *queries.Query,
) orgie.Options {
	return organizeFlags.GetOptions(
		repo.GetConfig().GetPrintOptions(),
		queries.GetTags(qg),
		repo.SkuFormatBoxCheckedOutNoColor(),
		repo.GetStore().GetAbbrStore().GetAbbr(),
		sku.ObjectFactory{},
	)
}

// PrepareOrganizeResultsForApply resolves results.Before/Original/
// WasGrouped/GroupingTags in place from results.After's real `_base` field
// and a fresh live requery -- dodder#374(b) plan §4's base-dereference and
// live-requery step. Extracted from LockAndCommitOrganizeResults so a
// caller with its OWN commit-plan construction (checkin.go's runOrganize,
// which folds organize's changes into one larger unified checkin commit
// rather than committing them separately) can get genuine base/live
// semantics for orgie.ChangesFromResults without also taking
// LockAndCommitOrganizeResults's own commit execution.
//
// results.After (the patch) is authoritative for `_base` regardless of
// what the caller populated results.Before/Original with at generation
// time, and results.QueryGroup must be non-nil (a real user query, the
// query a command like checkin/last/new already selected its objects
// with, or Organize.RunWithSkuType's WithExternalLike(skus) fallback) --
// re-running it against the live store is always meaningful "live" input
// for orgie.ComputeThreeWay (three_way.go).
func PrepareOrganizeResultsForApply(
	repo *local_working_copy.Repo,
	results orgie.OrganizeResults,
) (out orgie.OrganizeResults, err error) {
	out = results

	if out.Before, out.WasGrouped, out.GroupingTags, err = DereferenceOrganizeBase(
		repo,
		out.After,
	); err != nil {
		err = errors.Wrap(err)
		return out, err
	}

	live := sku.MakeSkuTypeSetMutable()

	var lock sync.Mutex

	if err = repo.GetStore().QueryTransactedAsSkuType(
		out.QueryGroup,
		func(sk sku.SkuType) (err error) {
			lock.Lock()
			defer lock.Unlock()

			cloned, _ := sk.Clone() //repool:owned
			return live.Add(cloned)
		},
	); err != nil {
		err = errors.Wrap(err)
		return out, err
	}

	out.Original = live

	// dodder#374(b) followup: a Base/Patch-touched object can fall out
	// of the tag-based query above (dodder's only selection mechanism
	// is tags, so a concurrent tag edit is enough) without ceasing to
	// exist -- dodder has no hard-delete. This ID-based fallback lets
	// ComputeThreeWay's live-drift check (three_way.go) still compare
	// against the object's true current state instead of silently
	// skipping it. Mirrors organize.go:93-96's established
	// Transacted->SkuType conversion idiom.
	out.FetchLiveById = func(
		objectId *ids.ObjectId,
	) (sku.SkuType, bool, error) {
		// store.ReadOneObjectId special-cases an empty objectId (e.g. a
		// still-unassigned id for a brand-new zettel proposal in Base)
		// by returning (nil, nil) -- no error, but also no object
		// (oscar/store/reader.go:150-152). Must be treated the same as
		// IsErrNotFound below, or CloneSkuTypeFromTransacted panics on
		// a nil src.
		transacted, err := repo.GetStore().ReadOneObjectId(objectId)
		if err != nil {
			if errors.IsErrNotFound(err) {
				return nil, false, nil
			}

			return nil, false, errors.Wrap(err)
		}

		if transacted == nil {
			return nil, false, nil
		}

		cloned, _ := sku.CloneSkuTypeFromTransacted( //repool:owned
			transacted,
			checked_out_state.Internal,
		)

		return cloned, true, nil
	}

	return out, err
}

// LockAndCommitOrganizeResults is the single funnel most organize commit
// paths go through (directly, or via OrganizeCommitFromReader) --
// PrepareOrganizeResultsForApply happens HERE, once, rather than in each
// of the CLI/MCP/Last/New call sites. checkin.go's runOrganize is the one
// caller that needs base/live resolution WITHOUT this function's own
// commit execution -- it calls PrepareOrganizeResultsForApply directly.
func LockAndCommitOrganizeResults(
	repo *local_working_copy.Repo,
	results orgie.OrganizeResults,
) (changeResults orgie.Changes, err error) {
	if results, err = PrepareOrganizeResultsForApply(repo, results); err != nil {
		err = errors.Wrap(err)
		return changeResults, err
	}

	if changeResults, err = orgie.ChangesFromResults(
		repo.GetConfig().GetPrintOptions(),
		results,
	); err != nil {
		err = errors.Wrap(err)
		return changeResults, err
	}

	count := changeResults.Changed.Len()

	if count > 30 {
		if !repo.Confirm(
			fmt.Sprintf(
				"a large number (%d) of objects are being changed. continue to commit?",
				count,
			),
			"",
		) {
			// TODO output organize file used
			errors.ContextCancelWith499ClientClosedRequest(repo)
			return changeResults, err
		}
	}

	var proto sku.Proto

	workspace := repo.GetEnvWorkspace()
	workspaceType := workspace.GetDefaults().GetDefaultType()

	proto.Metadata.GetTypeMutable().ResetWithType(workspaceType)

	builder := import_plan.MakeLocalBuilder()

	for _, changed := range changeResults.Changed.AllSkuAndIndex() {
		if err = builder.AddObject(changed.GetSkuExternal(), 0); err != nil {
			err = errors.Wrap(err)
			return changeResults, err
		}
	}

	plan, buildErr := builder.Build()
	if buildErr != nil {
		err = errors.Wrap(buildErr)
		return changeResults, err
	}

	plan.DefaultCommitOptions = sku.CommitOptions{
		Proto: proto,
		StoreOptions: sku.StoreOptions{
			AddToInventoryList: true,
			UpdateTai:          true,
			RunHooks:           true,
			Validate:           true,
			MergeCheckedOut:    true,
		},
	}

	if _, err = repo.ExecutePlan(plan); err != nil {
		err = errors.Wrap(err)
		return changeResults, err
	}

	return changeResults, err
}

func ApplyToOrganizeOptions(
	repo *local_working_copy.Repo,
	oo *orgie.Options,
) {
	oo.Config = repo.GetConfigPtr()
	oo.Abbr = repo.GetStore().GetAbbrStore().GetAbbr()

	if !repo.GetConfig().IsDryRun() {
		return
	}

	// AddPrototypeAndDirectiveOption, not AddPrototypeAndOption: RFC
	// 0015's merged two-plane revision reclassifies `_dry-run` to the
	// operational plane -- renders as "%:dry-run = true", not the
	// pre-RFC-0015 "- _dry-run = true" data-plane spelling.
	oo.SettingSet.AddPrototypeAndDirectiveOption(
		"dry-run",
		&orgie.SettingDryRun{
			MutableConfigDryRun: repo.GetConfigPtr(),
		},
	)
}
