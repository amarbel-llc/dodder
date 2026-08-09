package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/oscar/store"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

// MakeExpandedInventoryList builds the query-matched set like
// MakeInventoryList and then grows it into its transitive closure (types,
// tags, and other referenced objects) via the same expandEdges traversal the
// pull path uses, driven against the local stores (RFC-0008 §1). Traversal
// failures (e.g. a dangling blob reference in a mid-migration repo) are
// returned as skipped rather than erroring, so the caller can choose between
// pull's strict policy and transform's -skip_validation tolerance.
func (local *Repo) MakeExpandedInventoryList(
	query *queries.Query,
) (list *sku.HeapTransacted, skipped []error, err error) {
	if list, err = local.MakeInventoryList(query); err != nil {
		err = errors.Wrap(err)
		return list, skipped, err
	}

	explorer := store.MakeEdgeExplorer(
		local.GetObjectStore(),
		local.GetBlobStore(),
		local.GetEnvRepo(),
	)

	edges, err := expandEdges(list, local.GetObjectStore(), explorer)
	if err != nil {
		err = errors.Wrap(err)
		return list, skipped, err
	}

	skipped = edges.Skipped

	return list, skipped, err
}
