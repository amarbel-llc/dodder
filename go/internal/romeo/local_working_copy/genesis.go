package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/box_format"
	"code.linenisgreat.com/dodder/go/internal/golf/type_blobs"
	"code.linenisgreat.com/dodder/go/internal/hotel/import_plan"
	"code.linenisgreat.com/dodder/go/internal/hotel/inventory_list_coders"
	"code.linenisgreat.com/dodder/go/internal/india/config_log"
	"github.com/amarbel-llc/madder/go/pkgs/blob_store_id"
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/pool"
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
	builder := import_plan.MakeLocalBuilder()

	var toolBlobs toolBlobDigests

	if bigBang.IncludeDefaultPandocTools {
		if err = local.prepareToolTypes(&builder); err != nil {
			return errors.Wrap(err)
		}

		if toolBlobs, err = local.prepareToolBlobs(); err != nil {
			return errors.Wrap(err)
		}
	}

	var defaultTypeObjectId ids.TypeStruct

	if defaultTypeObjectId, err = local.prepareDefaultType(
		bigBang,
		&builder,
		toolBlobs,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	blobStoreId := local.GetEnvRepo().GetDefaultBlobStore().GetId()

	if !bigBang.BlobStoreId.IsEmpty() {
		blobStoreId = bigBang.BlobStoreId
	}

	blobStores := []blob_store_id.Id{blobStoreId}

	var configBlobId mad_domain_interfaces.MarklId
	var configType ids.Type
	var seededConfig bool

	if configBlobId, configType, seededConfig, err = local.prepareDefaultConfig(
		bigBang,
		blobStores,
		defaultTypeObjectId,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = local.prepareBuiltinActionableTypes(
		bigBang,
		&builder,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	plan, buildErr := builder.Build()
	if buildErr != nil {
		err = errors.Wrap(buildErr)
		return err
	}

	plan.DefaultCommitOptions = sku.CommitOptions{
		Proto:        local.GetStore().GetProtoZettel(),
		StoreOptions: sku.GetStoreOptionsCreate(),
	}

	if _, err = local.ExecutePlan(plan); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// Seed the repo-local config log with a root entry so fresh repos
	// read their config from the log at bootstrap (FDR 0020). No konfig
	// object is committed; config lives solely in the log. Runs after
	// ExecutePlan so the lock it acquired is released, and uses the
	// filesystem mutex directly (config_log.Append's "caller holds the
	// repo lock" contract is about that mutex, not the heavyweight repo
	// Unlock flush).
	if seededConfig {
		if err = local.seedConfigLog(configBlobId, configType); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

// seedConfigLog appends the genesis root entry to the repo-local config
// log under the repo filesystem lock. blobId is the default config TOML
// blob digest; configType is the config blob's own type
// (e.g. !toml-config-v2), which the entry keeps so store_config bootstrap
// can decode it via repo_configs.Coder.
func (local *Repo) seedConfigLog(
	blobId mad_domain_interfaces.MarklId,
	configType ids.Type,
) (err error) {
	box := box_format.MakeBoxTransactedArchive(
		local.GetEnv(),
		local.GetConfig().GetPrintOptions().WithPrintTai(true),
	)
	closet := inventory_list_coders.MakeCloset(local.GetEnvRepo(), box)

	cfgLog := config_log.Make(local.GetEnvRepo(), closet)

	lockSmith := local.GetEnvRepo().GetLockSmith()

	if err = lockSmith.Lock(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.Deferred(&err, lockSmith.Unlock)

	if err = cfgLog.Append(
		blobId,
		configType,
		local.GetStore().GetTai(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (local *Repo) prepareDefaultType(
	bigBang env_repo.BigBang,
	builder *import_plan.Builder,
	toolBlobs toolBlobDigests,
) (objectIdType ids.TypeStruct, err error) {
	if bigBang.ExcludeDefaultType {
		return objectIdType, err
	}

	objectIdType = ids.MustTypeStruct("md")
	tipe := ids.DefaultOrPanic(genres.Type)

	var blob type_blobs.TomlV2
	if bigBang.IncludeDefaultPandocTools {
		blob = type_blobs.DefaultWithPandocFormatter()
	} else {
		blob = type_blobs.Default()
	}

	object, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned

	if err = object.GetObjectIdMutable().SetWithId(objectIdType); err != nil {
		err = errors.Wrap(err)
		return objectIdType, err
	}

	var digest mad_domain_interfaces.MarklId

	if digest, _, err = local.GetStore().GetTypedBlobStore().Type.SaveBlobText(
		tipe,
		&blob,
	); err != nil {
		err = errors.Wrap(err)
		return objectIdType, err
	}

	object.GetMetadataMutable().GetBlobDigestMutable().ResetWithMarklId(digest)
	object.GetMetadataMutable().GetTypeMutable().ResetWithType(tipe)

	if bigBang.IncludeDefaultPandocTools {
		if err = addToolBlobReference(
			object, toolBlobs.commonFilter,
			"pandoc-lua_filter", "filters/dodder-common.lua",
		); err != nil {
			return objectIdType, errors.Wrap(err)
		}

		if err = addToolBlobReference(
			object, toolBlobs.editFilter,
			"pandoc-lua_filter", "filters/dodder-edit.lua",
		); err != nil {
			return objectIdType, errors.Wrap(err)
		}

		if err = addToolBlobReference(
			object, toolBlobs.editDefaults,
			"pandoc-defaults", "defaults/dodder-edit.yaml",
		); err != nil {
			return objectIdType, errors.Wrap(err)
		}
	}

	if err = builder.AddObject(object, 0); err != nil {
		err = errors.Wrap(err)
		return objectIdType, err
	}

	return objectIdType, err
}

// prepareBuiltinActionableTypes commits the !task and !chore built-in types
// when bigBang.IncludeBuiltinActionableTypes is set. Both types share the same
// field set (status, priority, due) declared in type_blobs.DefaultTaskType /
// DefaultChoreType. Opt-in for now per docs/plans/2026-04-06-task-type-genesis-
// and-haustoria-fields.md.
func (local *Repo) prepareBuiltinActionableTypes(
	bigBang env_repo.BigBang,
	builder *import_plan.Builder,
) (err error) {
	if !bigBang.IncludeBuiltinActionableTypes {
		return err
	}

	for _, builtin := range []struct {
		objectIdString string
		blob           type_blobs.TomlV2
	}{
		{
			objectIdString: "task",
			blob:           type_blobs.DefaultTaskType(),
		},
		{
			objectIdString: "chore",
			blob:           type_blobs.DefaultChoreType(),
		},
	} {
		objectIdType := ids.MustTypeStruct(builtin.objectIdString)
		tipe := ids.DefaultOrPanic(genres.Type)

		object, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned

		if err = object.GetObjectIdMutable().SetWithId(objectIdType); err != nil {
			err = errors.Wrap(err)
			return err
		}

		var digest mad_domain_interfaces.MarklId

		blob := builtin.blob
		if digest, _, err = local.GetStore().GetTypedBlobStore().Type.SaveBlobText(
			tipe,
			&blob,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		object.GetMetadataMutable().GetBlobDigestMutable().ResetWithMarklId(digest)
		object.GetMetadataMutable().GetTypeMutable().ResetWithType(tipe)

		if err = builder.AddObject(object, 0); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

// prepareDefaultConfig writes the default config blob and returns the
// blob digest and config type so the caller can seed the config log root
// entry after the plan commits. It creates NO konfig object: config lives
// solely in the config log (FDR 0020), and nothing reads a konfig object
// anymore. seeded is false when ExcludeDefaultConfig skips config genesis
// entirely (no log entry to seed).
func (local *Repo) prepareDefaultConfig(
	bigBang env_repo.BigBang,
	blobStores []blob_store_id.Id,
	defaultTypeObjectId ids.TypeStruct,
) (blobId mad_domain_interfaces.MarklId, configType ids.Type, seeded bool, err error) {
	if bigBang.ExcludeDefaultConfig {
		return blobId, configType, seeded, err
	}

	var typedBlob repo_configs.TypedBlob

	if blobId, typedBlob, err = writeDefaultMutableConfig(
		local,
		blobStores,
		defaultTypeObjectId,
	); err != nil {
		err = errors.Wrap(err)
		return blobId, configType, seeded, err
	}

	configType = ids.TypeStruct(typedBlob.Type).ToType()

	seeded = true

	return blobId, configType, seeded, err
}

func writeDefaultMutableConfig(
	repo *Repo,
	blobStores []blob_store_id.Id,
	defaultType ids.TypeStruct,
) (blobId mad_domain_interfaces.MarklId, typedBlob repo_configs.TypedBlob, err error) {
	typedBlob = repo_configs.DefaultOverlay(blobStores, defaultType)

	var blobWriter mad_domain_interfaces.BlobWriter

	if blobWriter, err = repo.GetEnvRepo().GetDefaultBlobStore().MakeBlobWriter(nil); err != nil {
		err = errors.Wrap(err)
		return blobId, typedBlob, err
	}

	defer errors.DeferredCloser(&err, blobWriter)

	bufferedWriter, repoolBufferedWriter := pool.GetBufferedWriter(blobWriter)
	defer repoolBufferedWriter()

	if _, err = repo_configs.Coder.Blob.EncodeTo(
		&typedBlob,
		bufferedWriter,
	); err != nil {
		err = errors.Wrap(err)
		return blobId, typedBlob, err
	}

	if err = bufferedWriter.Flush(); err != nil {
		err = errors.Wrap(err)
		return blobId, typedBlob, err
	}

	blobId = blobWriter.GetMarklId()

	return blobId, typedBlob, err
}
