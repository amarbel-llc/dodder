package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

const maxEdgeExpansionDepth = 5

func expandEdges(
	list *sku.HeapTransacted,
	objectStore sku.RepoStore,
	explorer sku.EdgeExplorer,
) (allEdges sku.Edges, err error) {
	if explorer == nil {
		return allEdges, nil
	}

	seen := make(map[string]struct{})
	seenBlobs := make(map[string]struct{})

	for object := range list.All() {
		seen[object.GetObjectId().String()] = struct{}{}
	}

	for range maxEdgeExpansionDepth {
		var pendingIds []ids.ObjectId

		for object := range list.All() {
			edges, exploreErr := explorer.ExploreEdges(object)
			if exploreErr != nil {
				return allEdges, errors.Wrap(exploreErr)
			}

			for _, oid := range edges.Objects {
				key := oid.String()
				if _, ok := seen[key]; !ok {
					seen[key] = struct{}{}
					pendingIds = append(pendingIds, oid)
				}
			}

			for _, blobId := range edges.Blobs {
				key := blobId.String()
				if _, ok := seenBlobs[key]; !ok {
					seenBlobs[key] = struct{}{}
					allEdges.Blobs = append(allEdges.Blobs, blobId)
				}
			}

			allEdges.Skipped = append(allEdges.Skipped, edges.Skipped...)
		}

		if len(pendingIds) == 0 {
			break
		}

		for i := range pendingIds {
			fetched, repool := sku.GetTransactedPool().GetWithRepool() //repool:suppress ownership transfer to list

			if err = objectStore.ReadOneInto(&pendingIds[i], fetched); err != nil {
				repool()

				if errors.IsErrNotFound(err) {
					err = nil
					continue
				}

				return allEdges, errors.Wrap(err)
			}

			allEdges.Objects = append(allEdges.Objects, pendingIds[i])

			if err = list.Add(fetched); err != nil {
				repool()
				return allEdges, errors.Wrap(err)
			}
		}
	}

	return allEdges, nil
}
