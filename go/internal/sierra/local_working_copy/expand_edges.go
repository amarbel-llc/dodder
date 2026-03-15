package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

const maxEdgeExpansionDepth = 5

func expandEdges(
	list *sku.HeapTransacted,
	objectStore sku.RepoStore,
) error {
	if objectStore == nil {
		return nil
	}

	seen := make(map[string]struct{})

	for object := range list.All() {
		seen[object.GetObjectId().String()] = struct{}{}
	}

	for depth := 0; depth < maxEdgeExpansionDepth; depth++ {
		var pendingIds []ids.ObjectId

		for object := range list.All() {
			if typeId := object.GetType(); !typeId.IsEmpty() && !ids.IsBuiltin(typeId) {
				key := typeId.String()
				if _, ok := seen[key]; !ok {
					seen[key] = struct{}{}
					var oid ids.ObjectId
					if err := oid.SetWithId(typeId); err != nil {
						return errors.Wrap(err)
					}
					pendingIds = append(pendingIds, oid)
				}
			}

			for tag := range object.AllTags() {
				key := tag.String()
				if _, ok := seen[key]; !ok {
					seen[key] = struct{}{}
					var oid ids.ObjectId
					if err := oid.SetWithId(tag); err != nil {
						return errors.Wrap(err)
					}
					pendingIds = append(pendingIds, oid)
				}
			}

			for ref := range object.GetMetadata().AllReferencedObjects() {
				key := ref.String()
				if _, ok := seen[key]; !ok {
					seen[key] = struct{}{}
					refCopy := ref
					pendingIds = append(pendingIds, refCopy)
				}
			}
		}

		if len(pendingIds) == 0 {
			break
		}

		for i := range pendingIds {
			fetched, repool := sku.GetTransactedPool().GetWithRepool()

			if err := objectStore.ReadOneInto(&pendingIds[i], fetched); err != nil {
				repool()

				if errors.IsErrNotFound(err) {
					continue
				}

				return errors.Wrap(err)
			}

			if err := list.Add(fetched); err != nil {
				repool()
				return errors.Wrap(err)
			}
		}
	}

	return nil
}
