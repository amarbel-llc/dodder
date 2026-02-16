package store

import (
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/collections_slice"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/catgut"
	"code.linenisgreat.com/dodder/go/src/charlie/expansion"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/foxtrot/tag_paths"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

// TODO extract into store_tags
func (store *Store) applyDormantAndRealizeTags(
	object *sku.Transacted,
) (err error) {
	ui.Log().Print("applying konfig to:", object)
	metadata := object.GetMetadataMutable()

	genre := genres.Must(object.GetGenre())
	isTag := genre == genres.Tag

	// if g.HasParents() {
	// 	k.SetHasChanges(fmt.Sprintf("adding etikett with parents: %s", sk))
	// }

	var tag ids.TagStruct

	// TODO better solution for "realizing" tags against Config.
	// Specifically, making this less fragile and dependent on remembering to do
	// ApplyToSku for each Sku. Maybe a factory?
	metadata.GetIndexMutable().GetTagPathsMutable().Reset()
	for tag := range metadata.AllTags() {
		metadata.GetIndexMutable().GetTagPathsMutable().AddTagOld(tag)
	}

	if isTag {
		objectIdString := object.ObjectId.String()

		if err = tag.Set(objectIdString); err != nil {
			err = errors.Wrap(err)
			return err
		}

		selfTag, _ := catgut.MakeFromString(objectIdString)
		object.GetMetadataMutable().GetIndexMutable().GetTagPaths().AddSelf(
			selfTag,
		)
	}

	if err = store.addSuperTags(object); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = store.addImplicitTags(object); err != nil {
		err = errors.Wrap(err)
		return err
	}

	object.SetDormant(store.dormantIndex.ContainsSku(object))

	return err
}

func (store *Store) addSuperTags(
	object *sku.Transacted,
) (err error) {
	genre := object.GetGenre()

	var expanded collections_slice.Slice[string]
	var objectIdString string

	switch genre {
	case genres.Tag, genres.Type, genres.Repo:
		objectIdString = object.ObjectId.String()

		quiter.AppendSeq(
			&expanded,
			expansion.ExpanderRight.Expand(objectIdString),
		)

	default:
		return err
	}

	for _, expandedObjectIdComponent := range expanded {
		if expandedObjectIdComponent == objectIdString ||
			expandedObjectIdComponent == "" {
			continue
		}

		func() {
			var tagOrRepoOrTypeObject *sku.Transacted

			if tagOrRepoOrTypeObject, err = store.storeConfig.GetConfig().GetTagOrRepoIdOrType(
				expandedObjectIdComponent,
			); err != nil {
				err = errors.Wrapf(
					err,
					"Expanded: %q",
					expandedObjectIdComponent,
				)
				return
			}

			if tagOrRepoOrTypeObject == nil {
				// this is ok because currently, konfig is applied twice.
				// However, this
				// is fragile as the order in which this method is called is
				// non-deterministic and the `GetTag` call may request an Tag we
				// have not processed yet
				return
			}

			tagPaths := tagOrRepoOrTypeObject.GetMetadata().GetIndex().GetTagPaths()

			if tagPaths.Paths.Len() <= 1 {
				return
			}

			prefix, _ := catgut.MakeFromString(expandedObjectIdComponent)

			newTagPaths := object.GetMetadataMutable().GetIndexMutable().GetTagPaths()

			if err = newTagPaths.AddSuperFrom(tagPaths, prefix); err != nil {
				err = errors.Wrap(err)
				return
			}
		}()
	}

	return err
}

func (store *Store) addImplicitTags(
	object *sku.Transacted,
) (err error) {
	metadata := object.GetMetadataMutable()
	tagSet := ids.MakeTagSetMutable()

	tagPaths := object.GetMetadataMutable().GetIndexMutable().GetTagPaths()

	addImplicitTags := func(tag ids.Tag) (err error) {
		tagPathWithType := tag_paths.MakePathWithType()
		tagPathWithType.Type = tag_paths.TypeIndirect
		tagStr, _ := catgut.MakeFromString(tag.String())
		tagPathWithType.Add(tagStr)

		implicitTags := store.storeConfig.GetConfig().GetImplicitTags(tag)

		if implicitTags.Len() == 0 {
			tagPaths.AddPathWithType(tagPathWithType)
			return err
		}

		for implicitTag := range implicitTags.All() {
			tagPathWithTypeClone := tagPathWithType.Clone()
			implicitTagStr, _ := catgut.MakeFromString(implicitTag.String())
			tagPathWithTypeClone.Add(implicitTagStr)

			tagPaths.AddPathWithType(tagPathWithTypeClone)
		}

		return err
	}

	for tag := range metadata.AllTags() {
		if err = addImplicitTags(tag); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	typeObject := store.storeConfig.GetConfig().GetApproximatedType(
		metadata.GetType(),
	).ApproximatedOrActual()

	if typeObject != nil {
		for tag := range typeObject.GetMetadata().AllTags() {
			ids.TagSetMutableAdd(tagSet, tag)
		}

		for tag := range typeObject.GetMetadata().AllTags() {
			if err = addImplicitTags(tag); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}
	}

	metadata.GetIndexMutable().SetImplicitTags(tagSet)

	return err
}
