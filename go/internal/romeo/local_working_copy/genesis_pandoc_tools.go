package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/type_blobs"
	"code.linenisgreat.com/dodder/go/internal/hotel/import_plan"
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

type toolBlobDigests struct {
	commonFilter        markl.Id
	editFilter          markl.Id
	renderFilter        markl.Id
	editDefaults        markl.Id
	renderDefaults      markl.Id
	htmlDefaults        markl.Id
	htmlPartialDefaults markl.Id
	gdocDefaults        markl.Id
	beamerDefaults      markl.Id
}

func (local *Repo) prepareToolTypes(
	builder *import_plan.Builder,
) (err error) {
	tipe := ids.DefaultOrPanic(genres.Type)

	prepareOne := func(objectIdString string, blob type_blobs.TomlV2) error {
		object, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned

		if err := object.GetObjectIdMutable().SetWithId(
			ids.MustTypeStruct(objectIdString),
		); err != nil {
			return errors.Wrap(err)
		}

		var digest mad_domain_interfaces.MarklId

		if digest, _, err = local.GetStore().GetTypedBlobStore().Type.SaveBlobText(
			tipe,
			&blob,
		); err != nil {
			return errors.Wrap(err)
		}

		object.GetMetadataMutable().GetBlobDigestMutable().ResetWithMarklId(digest)
		object.GetMetadataMutable().GetTypeMutable().ResetWithType(tipe)

		if err := builder.AddObject(object, 0); err != nil {
			return errors.Wrap(err)
		}

		return nil
	}

	if err = prepareOne("pandoc-defaults", type_blobs.DefaultPandocDefaults()); err != nil {
		return errors.Wrap(err)
	}

	if err = prepareOne("pandoc-lua_filter", type_blobs.DefaultPandocLuaFilter()); err != nil {
		return errors.Wrap(err)
	}

	return err
}

func (local *Repo) writeRawBlob(content []byte) (digest markl.Id, err error) {
	var writer mad_domain_interfaces.BlobWriter

	if writer, err = local.GetEnvRepo().GetDefaultBlobStore().MakeBlobWriter(nil); err != nil {
		return digest, errors.Wrap(err)
	}

	defer errors.DeferredCloser(&err, writer)

	if _, err = writer.Write(content); err != nil {
		return digest, errors.Wrap(err)
	}

	digest.ResetWithMarklId(writer.GetMarklId())

	return digest, err
}

func (local *Repo) prepareToolBlobs() (digests toolBlobDigests, err error) {
	for _, blob := range []struct {
		digest  *markl.Id
		content []byte
	}{
		{&digests.commonFilter, embeddedPandocCommonFilter},
		{&digests.editFilter, embeddedPandocEditFilter},
		{&digests.renderFilter, embeddedPandocRenderFilter},
		{&digests.editDefaults, embeddedPandocEditDefaults},
		{&digests.renderDefaults, embeddedPandocRenderDefaults},
		{&digests.htmlDefaults, embeddedPandocHtmlDefaults},
		{&digests.htmlPartialDefaults, embeddedPandocHtmlPartialDefaults},
		{&digests.gdocDefaults, embeddedPandocGdocDefaults},
		{&digests.beamerDefaults, embeddedPandocBeamerDefaults},
	} {
		if *blob.digest, err = local.writeRawBlob(blob.content); err != nil {
			return digests, errors.Wrap(err)
		}
	}

	return digests, err
}

// attachPandocToolRefs attaches the blob-backed pandoc tool references (the
// three lua filters plus the edit/render/html/html-partial/gdoc/beamer
// defaults) to object. Shared by prepareDefaultType (!md) and
// prepareBuiltinActionableTypes (!task/!chore/!habit); callers gate it on
// !bigBang.ExcludeDefaultPandocTools so the referenced tool blobs always
// exist. The actionable types carry the beamer/render defaults blobs even
// though they lack the corresponding formatters: a single shared ref set
// keeps genesis simple and the extra refs are harmless.
func attachPandocToolRefs(
	object *sku.Transacted,
	toolBlobs toolBlobDigests,
) (err error) {
	for _, ref := range []struct {
		digest     markl.Id
		typeString string
		alias      string
	}{
		{toolBlobs.commonFilter, "pandoc-lua_filter", "filters/dodder-common.lua"},
		{toolBlobs.editFilter, "pandoc-lua_filter", "filters/dodder-edit.lua"},
		{toolBlobs.renderFilter, "pandoc-lua_filter", "filters/dodder-render.lua"},
		{toolBlobs.editDefaults, "pandoc-defaults", "defaults/dodder-edit.yaml"},
		{toolBlobs.renderDefaults, "pandoc-defaults", "defaults/dodder-render.yaml"},
		{toolBlobs.htmlDefaults, "pandoc-defaults", "defaults/dodder-html.yaml"},
		{toolBlobs.htmlPartialDefaults, "pandoc-defaults", "defaults/dodder-html-partial.yaml"},
		{toolBlobs.gdocDefaults, "pandoc-defaults", "defaults/dodder-gdoc.yaml"},
		{toolBlobs.beamerDefaults, "pandoc-defaults", "defaults/dodder-beamer.yaml"},
	} {
		if err = addToolBlobReference(
			object, ref.digest, ref.typeString, ref.alias,
		); err != nil {
			return errors.Wrap(err)
		}
	}

	return err
}

func addToolBlobReference(
	object *sku.Transacted,
	digest markl.Id,
	typeString string,
	alias string,
) (err error) {
	var typeLock markl.Lock[ids.SeqId, *ids.SeqId]
	marshaler := markl.MakeMutableLockCoderValueNotRequired(&typeLock)

	if err = marshaler.Set(ids.MakeTypeString(typeString)); err != nil {
		return errors.Wrap(err)
	}

	metadata := object.GetMetadataMutable()
	metadata.AddBlobReference(digest, typeLock)

	return metadata.SetBlobReferenceAlias(digest, alias)
}
