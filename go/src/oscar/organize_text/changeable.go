package organize_text

import (
	"fmt"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/foxtrot/tag_paths"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

var keyer = sku.GetExternalLikeKeyer[sku.SkuType]()

func (ot *Text) GetSkus(
	original sku.SkuTypeSet,
) (out SkuMapWithOrder, err error) {
	out = MakeSkuMapWithOrder(original.Len())

	if err = ot.addToSet(
		ot,
		out,
		original,
	); err != nil {
		err = errors.Wrap(err)
		return out, err
	}

	return out, err
}

// TODO: claude: refactor to use single call to `GetMetadataMutable()` at start
// of loops to simplify subsequent nested method calls
func (assignment *Assignment) addToSet(
	ot *Text,
	output SkuMapWithOrder,
	objectsFromBefore sku.SkuTypeSet,
) (err error) {
	expanded := ids.MakeTagSetMutable()

	if err = assignment.AllTags(expanded); err != nil {
		err = errors.Wrap(err)
		return err
	}

	for _, organizeObject := range assignment.All() {
		var outputObject sku.SkuType

		objectKey := keyer.GetKey(organizeObject.sku)

		previouslyProcessedObject, wasPreviouslyProcessed := output.m[objectKey]

		if !wasPreviouslyProcessed {
			outputObject, _ = ot.ObjectFactory.GetWithRepool()

			ot.ObjectFactory.ResetWith(outputObject, organizeObject.sku)

			if !ot.Metadata.Type.IsEmpty() {
				outputObject.GetSkuExternal().GetMetadataMutable().GetTypeMutable().ResetWithObjectId(
					ot.Metadata.Type,
				)
			}

			outputObject.GetSkuExternal().RepoId.ResetWith(ot.Metadata.RepoId)

			output.Add(outputObject)

			objectOriginal, hasOriginal := objectsFromBefore.Get(objectKey)

			if hasOriginal {
				outputObject.GetSkuExternal().GetMetadataMutable().GetBlobDigestMutable().ResetWithMarklId(
					objectOriginal.GetSkuExternal().GetMetadata().GetBlobDigest(),
				)

				outputObject.GetSkuExternal().GetMetadataMutable().GetTypeMutable().ResetWith(
					objectOriginal.GetSkuExternal().GetMetadata().GetType(),
				)

				outputObject.GetSkuExternal().GetSkuExternal().GetMetadataMutable().GetBlobDigestMutable().ResetWithMarklId(
					objectOriginal.GetSkuExternal().GetSkuExternal().GetMetadata().GetBlobDigest(),
				)

				outputObject.GetSkuExternal().GetSkuExternal().GetMetadataMutable().GetTypeMutable().ResetWith(
					objectOriginal.GetSkuExternal().GetSkuExternal().GetMetadata().GetType(),
				)

				outputObject.SetState(objectOriginal.GetState())

				{
					src := objectOriginal.GetSkuExternal().GetMetadata()

					outputObject.GetSkuExternal().GetMetadataMutable().GetIndexMutable().GetFieldsMutable().ResetWithSeq(
						src.GetIndex().GetFields(),
					)
				}
			}

			outputMetadata := outputObject.GetSkuExternal().GetMetadataMutable()

			for tag := range ot.Metadata.All() {
				if organizeObject.tipe == tag_paths.TypeUnknown {
					continue
				}

				if _, ok := outputMetadata.GetIndex().GetTagPaths().All.ContainsString(
					tag.String(),
				); ok {
					continue
				}

				outputObject.GetSkuExternal().AddTag(tag)
			}

			if !ot.Metadata.Type.IsEmpty() {
				outputObject.GetSkuExternal().GetMetadataMutable().GetTypeMutable().ResetWithType(
					ot.Metadata.Type,
				)
			}
		} else {
			outputObject = previouslyProcessedObject.sku
		}

		if organizeObject.GetSkuExternal().ObjectId.String() == "" {
			panic(fmt.Sprintf("%s: object id is nil", organizeObject))
		}

		if outputObject == nil {
			panic("empty object")
		}

		if err = outputObject.GetSkuExternal().GetMetadataMutable().GetDescriptionMutable().Set(
			organizeObject.GetSkuExternal().GetMetadata().GetDescription().String(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if !organizeObject.GetSkuExternal().GetMetadata().GetType().IsEmpty() {
			if err = outputObject.GetSkuExternal().GetMetadataMutable().GetTypeMutable().SetType(
				organizeObject.GetSkuExternal().GetMetadata().GetType().String(),
			); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}

		if !organizeObject.tipe.IsDirectOrSelf() {
			return err
		}

		quiter.AppendSeq(
			outputObject.GetSkuExternal().GetMetadataMutable().GetIndexMutable().GetCommentsMutable(),
			organizeObject.GetSkuExternal().GetMetadata().GetIndex().GetComments(),
		)

		for tag := range organizeObject.GetSkuExternal().GetMetadata().AllTags() {
			if err = outputObject.GetSkuExternal().AddTag(tag); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}

		for tag := range expanded.All() {
			outputObject.GetSkuExternal().AddTag(tag)
		}
	}

	for _, c := range assignment.Children {
		if err = c.addToSet(ot, output, objectsFromBefore); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}
