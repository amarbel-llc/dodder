package repo_actions

import (
	"fmt"
	"sync"

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

// LockAndCommitOrganizeResults is the single funnel all organize commit
// paths go through (directly, or via OrganizeCommitFromReader) --
// dodder#374(b) plan §4's base-dereference and live-requery happen HERE,
// once, rather than in each of the CLI/MCP/Last/New call sites: results.After
// (the patch) is authoritative for `_base` regardless of what the caller
// populated results.Before/Original with at generation time, and
// results.QueryGroup is always non-nil by this point (a real user query, or
// Organize.RunWithSkuType's WithExternalLike(skus) fallback for the
// Last/New flows), so re-running it against the live store is always
// meaningful "live" input for orgie.ComputeThreeWay (three_way.go).
func LockAndCommitOrganizeResults(
	repo *local_working_copy.Repo,
	results orgie.OrganizeResults,
) (changeResults orgie.Changes, err error) {
	if results.Before, results.WasGrouped, results.GroupingTags, err = DereferenceOrganizeBase(
		repo,
		results.After,
	); err != nil {
		err = errors.Wrap(err)
		return changeResults, err
	}

	live := sku.MakeSkuTypeSetMutable()

	var lock sync.Mutex

	if err = repo.GetStore().QueryTransactedAsSkuType(
		results.QueryGroup,
		func(sk sku.SkuType) (err error) {
			lock.Lock()
			defer lock.Unlock()

			cloned, _ := sk.Clone() //repool:owned
			return live.Add(cloned)
		},
	); err != nil {
		err = errors.Wrap(err)
		return changeResults, err
	}

	results.Original = live

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

	oo.AddPrototypeAndOption(
		"dry-run",
		&orgie.OptionCommentDryRun{
			MutableConfigDryRun: repo.GetConfigPtr(),
		},
	)
}
