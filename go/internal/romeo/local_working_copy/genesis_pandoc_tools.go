package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/type_blobs"
	"code.linenisgreat.com/dodder/go/internal/hotel/import_plan"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

type toolBlobDigests struct {
	commonFilter markl.Id
	editFilter   markl.Id
	editDefaults markl.Id
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

		var digest domain_interfaces.MarklId

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
	var writer domain_interfaces.BlobWriter

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
	if digests.commonFilter, err = local.writeRawBlob(embeddedPandocCommonFilter); err != nil {
		return digests, errors.Wrap(err)
	}

	if digests.editFilter, err = local.writeRawBlob(embeddedPandocEditFilter); err != nil {
		return digests, errors.Wrap(err)
	}

	if digests.editDefaults, err = local.writeRawBlob(embeddedPandocEditDefaults); err != nil {
		return digests, errors.Wrap(err)
	}

	return digests, err
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
