package store_fs

import (
	"maps"
	"slices"

	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter_set"
	"code.linenisgreat.com/dodder/go/lib/delta/collections_value"
)

type fsItemData struct {
	interfaces.SetMutable[*sku.FSItem]
	digests map[string]interfaces.SetMutable[*sku.FSItem]
}

func makeFSItemData() fsItemData {
	return fsItemData{
		SetMutable: collections_value.MakeMutableValueSet[*sku.FSItem](nil),
		digests:    make(map[string]interfaces.SetMutable[*sku.FSItem]),
	}
}

func (data *fsItemData) Clone() (dst fsItemData) {
	dst.SetMutable = collections_value.MakeMutableValueSet(
		nil,
		slices.Collect(data.SetMutable.All())...,
	)

	dst.digests = maps.Clone(data.digests)
	return dst
}

func (data *fsItemData) ConsolidateDuplicateBlobs() (err error) {
	replacement := collections_value.MakeMutableValueSet[*sku.FSItem](nil)

	for _, fds := range data.digests {
		if fds.Len() == 1 {
			replacement.Add(quiter_set.Any(fds))
		}

		sorted := quiter.ElementsSorted(
			fds,
			func(a, b *sku.FSItem) bool {
				return a.GetExternalObjectId().String() < b.GetExternalObjectId().String()
			},
		)

		top := sorted[0]

		for _, other := range sorted[1:] {
			for item := range other.FDs.All() {
				top.FDs.Add(item)
			}
		}

		replacement.Add(top)
	}

	// TODO make less leaky
	data.SetMutable = replacement

	return err
}
