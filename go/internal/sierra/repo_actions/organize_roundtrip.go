package repo_actions

import (
	"io"
	"sync"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/kilo/orgie"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter_set"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

// orgie-extract: this organize plan/commit round-trip is the shared seam used
// by both the CLI `organize` command and the MCP organize_plan/organize_commit
// tools (#7). It is a prime candidate to move into amarbel-llc/orgie alongside
// the orgie package; future work (#3) adds a structured (JSON / box-format)
// alternative to the text round-trip, and object-signature drift detection
// between plan and commit. Keep CLI and MCP on these two functions so there is
// one place to evolve.

// OrganizePlan resolves a query into the objects it matches, builds the
// CreateOrganizeFile op (Skus, Type, TagSet) the way the CLI `organize` command
// does, and renders the "before" organize-text tree. It returns the rendered
// tree (the baseline an edited buffer is diffed against) and the original
// object set. Read-only: no store mutation.
// The returned effective query group is the one objects were rendered against
// (with any default-query substitution applied); callers that later commit an
// edited buffer must diff against this same group, so OrganizeCommitFromReader
// takes it rather than the caller's pre-substitution query.
func OrganizePlan(
	repo *local_working_copy.Repo,
	queryGroup *queries.Query,
	flags orgie.Flags,
) (
	before *orgie.Text,
	original sku.SkuTypeSetMutable,
	effective *queries.Query,
	err error,
) {
	original = sku.MakeSkuTypeSetMutable()

	var lock sync.Mutex

	// Collect objects against the query as given (before any default-query
	// substitution): an empty query collects nothing, so an unqueried organize
	// in a workspace renders just the default-tag header, not every object
	// (workspace.bats workspace_organize).
	if err = repo.GetStore().QueryTransactedAsSkuType(
		queryGroup,
		func(checkedOut sku.SkuType) (err error) {
			lock.Lock()
			defer lock.Unlock()

			cloned, _ := checkedOut.Clone() //repool:owned
			return original.Add(cloned)
		},
	); err != nil {
		err = errors.Wrap(err)
		return before, original, queryGroup, err
	}

	// After collection, fall back to the workspace default query for the op's
	// tag/grouping derivation (matches the CLI organize ordering).
	if defaultQuery := queryGroup.GetDefaultQuery(); queryGroup.IsEmpty() &&
		defaultQuery != nil {
		queryGroup = defaultQuery
	}

	effective = queryGroup

	createOrganizeFileOp := MakeCreateOrganizeFile(
		repo,
		MakeOrganizeOptionsWithQueryGroup(repo, flags, queryGroup),
	)

	createOrganizeFileOp.Skus = original

	types := queries.GetTypes(queryGroup)
	if types.Len() == 1 {
		createOrganizeFileOp.Type = quiter_set.Any(types)
	}

	tags := queries.GetTags(queryGroup)

	// With no matching objects, seed the heading tags from the workspace
	// defaults so an empty buffer still offers somewhere to add objects under
	// (matches the CLI behavior).
	if original.Len() == 0 {
		workspaceTags := repo.GetEnvWorkspace().GetDefaults().GetDefaultTags()
		for tag := range workspaceTags.All() {
			ids.TagSetMutableAdd(tags, tag)
		}
	}

	createOrganizeFileOp.TagSet = tags

	if before, err = createOrganizeFileOp.Run(); err != nil {
		err = errors.Wrap(err)
		return before, original, effective, err
	}

	return before, original, effective, err
}

// OrganizeCommitFromReader parses an edited organize buffer from r (the "after"
// tree) and commits the resulting tag/description/move changes against the
// before/original baseline produced by OrganizePlan. Returns the applied
// changes. Mutating: acquires the repo lock via LockAndCommitOrganizeResults.
func OrganizeCommitFromReader(
	repo *local_working_copy.Repo,
	queryGroup *queries.Query,
	before *orgie.Text,
	original sku.SkuTypeSet,
	r io.Reader,
) (changes orgie.Changes, err error) {
	var after *orgie.Text

	if after, err = MakeReadOrganizeFile(repo).Run(
		r,
		orgie.NewMetadata(queryGroup.RepoId),
	); err != nil {
		err = errors.Wrap(err)
		return changes, err
	}

	if changes, err = LockAndCommitOrganizeResults(
		repo,
		orgie.OrganizeResults{
			Before:     before,
			After:      after,
			Original:   original,
			QueryGroup: queryGroup,
		},
	); err != nil {
		err = errors.Wrap(err)
		return changes, err
	}

	return changes, err
}
