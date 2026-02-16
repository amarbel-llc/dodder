package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/golf/repo_configs"
	"code.linenisgreat.com/dodder/go/src/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/lima/type_blobs"
)

func Genesis(
	bigBang env_repo.BigBang,
	envRepo env_repo.Env,
) (repo *Repo) {
	repo = MakeWithEnvRepo(OptionsEmpty, envRepo)

	if err := repo.dormantIndex.Flush(
		repo.GetEnvRepo(),
		repo.PrinterHeader(),
		repo.config.GetConfig().IsDryRun(),
	); err != nil {
		repo.Cancel(err)
	}

	repo.Must(errors.MakeFuncContextFromFuncErr(repo.Reset))
	repo.Must(errors.MakeFuncContextFromFuncErr(repo.envRepo.ResetCache))

	if err := repo.initDefaultTypeAndConfig(bigBang); err != nil {
		repo.Cancel(err)
	}

	repo.Must(errors.MakeFuncContextFromFuncErr(repo.Lock))
	repo.Must(errors.MakeFuncContextFromFuncErr(repo.GetStore().ResetIndexes))
	repo.Must(errors.MakeFuncContextFromFuncErr(repo.Unlock))

	return repo
}

func (local *Repo) initDefaultTypeAndConfig(
	bigBang env_repo.BigBang,
) (err error) {
	// TODO determine if this lock/unlock is necessary
	if err = local.Lock(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.Deferred(&err, local.Unlock)

	var defaultTypeObjectId ids.TypeStruct

	if defaultTypeObjectId, err = local.initDefaultTypeIfNecessaryAfterLock(
		bigBang,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	blobStoreId := local.GetEnvRepo().GetDefaultBlobStore().GetId()

	if !bigBang.BlobStoreId.IsEmpty() {
		blobStoreId = bigBang.BlobStoreId
	}

	if err = local.initDefaultConfigIfNecessaryAfterLock(
		bigBang,
		blobStoreId,
		defaultTypeObjectId,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (local *Repo) initDefaultTypeIfNecessaryAfterLock(
	bigBang env_repo.BigBang,
) (objectIdType ids.TypeStruct, err error) {
	if bigBang.ExcludeDefaultType {
		return objectIdType, err
	}

	objectIdType = ids.MustTypeStruct("md")
	tipe := ids.DefaultOrPanic(genres.Type)
	blob := type_blobs.Default()

	object, objectRepool := sku.GetTransactedPool().GetWithRepool()
	defer objectRepool()

	if err = object.ObjectId.SetWithId(objectIdType); err != nil {
		err = errors.Wrap(err)
		return objectIdType, err
	}

	var objectId ids.ObjectId

	if err = objectId.SetWithId(objectIdType); err != nil {
		err = errors.Wrap(err)
		return objectIdType, err
	}

	var digest domain_interfaces.MarklId

	if digest, _, err = local.GetStore().GetTypedBlobStore().Type.SaveBlobText(
		tipe,
		&blob,
	); err != nil {
		err = errors.Wrap(err)
		return objectIdType, err
	}

	object.GetMetadataMutable().GetBlobDigestMutable().ResetWithMarklId(digest)
	object.GetMetadataMutable().GetTypeMutable().ResetWithType(tipe)

	if err = local.GetStore().CreateOrUpdateDefaultProto(
		object,
		sku.GetStoreOptionsCreate(),
	); err != nil {
		err = errors.Wrap(err)
		return objectIdType, err
	}

	return objectIdType, err
}

func (local *Repo) initDefaultConfigIfNecessaryAfterLock(
	bigBang env_repo.BigBang,
	defaultBlobStoreId blob_store_id.Id,
	defaultTypeObjectId ids.TypeStruct,
) (err error) {
	if bigBang.ExcludeDefaultConfig {
		return err
	}

	var blobId domain_interfaces.MarklId
	var typedBlob repo_configs.TypedBlob

	if blobId, typedBlob, err = writeDefaultMutableConfig(
		local,
		defaultBlobStoreId,
		defaultTypeObjectId,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	newConfig, _ := sku.GetTransactedPool().GetWithRepool()

	if err = newConfig.ObjectId.SetWithId(
		ids.Config,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = newConfig.SetBlobDigest(blobId); err != nil {
		err = errors.Wrap(err)
		return err
	}

	newConfig.GetMetadataMutable().GetTypeMutable().ResetWithType(typedBlob.Type)

	if err = local.GetStore().CreateOrUpdateDefaultProto(
		newConfig,
		sku.GetStoreOptionsCreate(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func writeDefaultMutableConfig(
	repo *Repo,
	defaultBlobStoreId blob_store_id.Id,
	defaultType ids.TypeStruct,
) (blobId domain_interfaces.MarklId, typedBlob repo_configs.TypedBlob, err error) {
	typedBlob = repo_configs.DefaultOverlay(defaultBlobStoreId, defaultType)

	coder := repo.GetStore().GetConfigBlobCoder()

	var blobWriter domain_interfaces.BlobWriter

	if blobWriter, err = repo.GetEnvRepo().GetDefaultBlobStore().MakeBlobWriter(nil); err != nil {
		err = errors.Wrap(err)
		return blobId, typedBlob, err
	}

	defer errors.DeferredCloser(&err, blobWriter)

	if _, err = coder.EncodeTo(
		&typedBlob,
		blobWriter,
	); err != nil {
		err = errors.Wrap(err)
		return blobId, typedBlob, err
	}

	blobId = blobWriter.GetMarklId()

	return blobId, typedBlob, err
}
