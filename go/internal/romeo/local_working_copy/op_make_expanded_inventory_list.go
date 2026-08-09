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
// pull path uses, driven against the local stores (RFC-0008 §1).
func (local *Repo) MakeExpandedInventoryList(
	query *queries.Query,
) (list *sku.HeapTransacted, err error) {
	if list, err = local.MakeInventoryList(query); err != nil {
		err = errors.Wrap(err)
		return list, err
	}

	explorer := store.MakeEdgeExplorer(
		local.GetObjectStore(),
		local.GetBlobStore(),
		local.GetEnvRepo(),
	)

	edges, err := expandEdges(list, local.GetObjectStore(), explorer)
	if err != nil {
		err = errors.Wrap(err)
		return list, err
	}

	if len(edges.Skipped) > 0 {
		err = errors.Errorf(
			"edge traversal had %d failures: %s",
			len(edges.Skipped),
			edges.Skipped[0],
		)
		return list, err
	}

	return list, err
}
