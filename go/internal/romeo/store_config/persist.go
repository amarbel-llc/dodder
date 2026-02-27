package store_config

import (
	"encoding/gob" // TODO remove
	"os"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/values"
	"code.linenisgreat.com/dodder/go/internal/charlie/collections_value"
	"code.linenisgreat.com/dodder/go/internal/charlie/files"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/lima/type_blobs"
	"code.linenisgreat.com/dodder/go/internal/mike/typed_blob_store"
)

func init() {
	gob.Register(repo_configs.V1{})
	gob.Register(repo_configs.V0{})
}

func (store *store) recompile(
	blobStore typed_blob_store.Stores,
) (err error) {
	if err = store.recompileTags(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = store.recompileTypes(blobStore); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *store) recompileTags() (err error) {
	store.config.ImplicitTags = make(implicitTagMap)

	for tagObject := range store.config.Tags.All() {
		var tag ids.TagStruct

		if err = tag.Set(tagObject.String()); err != nil {
			err = errors.Wrapf(
				err,
				"Sku: %s",
				sku.StringTaiGenreObjectIdObjectDigestBlobDigest(
					&tagObject.Transacted,
				),
			)
			return err
		}

		if err = store.config.AccumulateImplicitTags(tag); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (store *store) recompileTypes(
	blobStore typed_blob_store.Stores,
) (err error) {
	inlineTypes := collections_value.MakeMutableValueSet[values.String](nil)

	defer func() {
		store.config.InlineTypes = collections_value.MakeValueSet(
			nil,
			inlineTypes.All(),
		)
	}()

	for tagObject := range store.config.Types.All() {
		tipe := tagObject.GetSku().GetType()
		var commonBlob type_blobs.Blob
		var repool interfaces.FuncRepool

		if commonBlob, repool, _, err = blobStore.Type.ParseTypedBlob(
			tipe,
			tagObject.GetBlobDigest(),
		); err != nil {
			err = errors.Wrap(err)
			return err
		}

		defer repool()

		if commonBlob == nil {
			err = errors.ErrorWithStackf(
				"nil type blob for type: %q. Sku: %s",
				tipe,
				tagObject,
			)
			return err
		}

		fileExtension := commonBlob.GetFileExtension()

		if fileExtension == "" {
			fileExtension = tagObject.GetObjectId().ToType().StringSansOp()
		}

		// TODO-P2 enforce uniqueness
		store.config.ExtensionsToTypes[fileExtension] = tagObject.GetObjectId().String()
		store.config.TypesToExtensions[tagObject.GetObjectId().String()] = fileExtension

		isBinary := commonBlob.GetBinary()
		if !isBinary {
			inlineTypes.Add(values.MakeString(tagObject.ObjectId.String()))
		}

	}
	return err
}

func (store *store) HasChanges() (ok bool) {
	store.config.lock.Lock()
	defer store.config.lock.Unlock()

	ok = len(store.config.compiled.changes) > 0

	if ok {
		ui.Log().Print(store.config.compiled.changes)
	}

	return ok
}

func (store *store) GetChanges() (out []string) {
	store.config.lock.Lock()
	defer store.config.lock.Unlock()

	out = make([]string, len(store.config.changes))
	copy(out, store.config.changes)

	return out
}

func (compiled *compiled) SetNeedsRecompile(reason string) {
	compiled.lock.Lock()
	defer compiled.lock.Unlock()

	compiled.setNeedsRecompile(reason)
}

func (compiled *compiled) setNeedsRecompile(reason string) {
	compiled.changes = append(compiled.changes, reason)
}

func (store *store) loadMutableConfig(
	envRepo env_repo.Env,
) (err error) {
	var file *os.File

	path := envRepo.FileConfig()

	if file, err = files.Open(path); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, file)

	dec := gob.NewDecoder(file)

	if err = dec.Decode(&store.config.compiled); err != nil {
		if errors.IsEOF(err) {
			err = nil
		} else {
			err = errors.Wrap(err)
		}

		return err
	}

	if err = store.loadMutableConfigBlob(
		store.config.Sku.GetType().ToType(),
		store.config.Sku.GetBlobDigest(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *store) Flush(
	envRepo env_repo.Env,
	blobStore typed_blob_store.Stores,
	printerHeader interfaces.FuncIter[string],
) (err error) {
	if !store.HasChanges() || store.config.IsDryRun() {
		return err
	}

	waitGroup := errors.MakeWaitGroupParallel()
	waitGroup.Do(func() (err error) {
		if err = store.flushMutableConfig(envRepo, blobStore, printerHeader); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	})

	if err = waitGroup.GetError(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	store.config.changes = store.config.changes[:0]

	return err
}

func (store *store) flushMutableConfig(
	envRepo env_repo.Env,
	blobStore typed_blob_store.Stores,
	printerHeader interfaces.FuncIter[string],
) (err error) {
	if err = printerHeader("recompiling konfig"); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = store.recompile(blobStore); err != nil {
		err = errors.Wrap(err)
		return err
	}

	path := envRepo.FileConfig()

	var file *os.File

	if file, err = files.OpenCreateWriteOnlyTruncate(path); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, file)

	enc := gob.NewEncoder(file)

	if err = enc.Encode(&store.config.compiled); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = printerHeader("recompiled konfig"); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *store) loadMutableConfigBlob(
	mutableConfigType ids.TypeStruct,
	blobId domain_interfaces.MarklId,
) (err error) {
	var blobReader domain_interfaces.BlobReader

	if blobReader, err = store.envRepo.GetDefaultBlobStore().MakeBlobReader(
		blobId,
	); err != nil {
		ui.Debug().PrintDebug(store.envRepo.GetXDG())
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, blobReader)

	typedBlob := repo_configs.TypedBlob{
		Type: mutableConfigType,
	}

	if _, err = repo_configs.Coder.DecodeFrom(
		&typedBlob,
		blobReader,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	store.config.configRepo = typedBlob.Blob

	return err
}
