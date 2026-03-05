package store_config

import (
	"encoding/gob" // TODO remove once V13 support dropped
	"os"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/store_version"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/golf/env_repo"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/type_blobs"
	"code.linenisgreat.com/dodder/go/internal/india/stream_index"
	"code.linenisgreat.com/dodder/go/internal/juliett/typed_blob_store"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"code.linenisgreat.com/dodder/go/lib/charlie/values"
	"code.linenisgreat.com/dodder/go/lib/delta/collections_value"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
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
			inlineTypes.Add(values.MakeString(tagObject.GetObjectId().String()))
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
	if store_version.GreaterOrEqual(
		envRepo.GetStoreVersion(),
		store_version.V14,
	) {
		return store.loadMutableConfigStreamIndex(envRepo)
	}

	return store.loadMutableConfigGob(envRepo)
}

func (store *store) loadMutableConfigGob(
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

func (store *store) loadMutableConfigStreamIndex(
	envRepo env_repo.Env,
) (err error) {
	var coder stream_index.ListCoder

	if err = store.loadStreamIndexFile(
		envRepo.FileConfig(),
		&coder,
		func(object *sku.Transacted) error {
			sku.Resetter.ResetWith(&store.config.Sku, object)
			return nil
		},
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = store.loadStreamIndexFile(
		envRepo.FileConfigTags(),
		&coder,
		func(object *sku.Transacted) error {
			var t tag
			sku.Resetter.ResetWith(&t.Transacted, object)
			store.config.Tags.Add(&t)
			return nil
		},
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = store.loadStreamIndexFile(
		envRepo.FileConfigTypes(),
		&coder,
		func(object *sku.Transacted) error {
			b, _ := sku.GetTransactedPool().GetWithRepool()
			sku.Resetter.ResetWith(b, object)
			store.config.Types.Add(b)
			return nil
		},
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = store.loadStreamIndexFile(
		envRepo.FileConfigRepos(),
		&coder,
		func(object *sku.Transacted) error {
			b, _ := sku.GetTransactedPool().GetWithRepool()
			sku.Resetter.ResetWith(b, object)
			store.config.Repos.Add(b)
			return nil
		},
	); err != nil {
		err = errors.Wrap(err)
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

func (store *store) loadStreamIndexFile(
	path string,
	coder *stream_index.ListCoder,
	each func(*sku.Transacted) error,
) (err error) {
	var file *os.File

	if file, err = files.Open(path); err != nil {
		if errors.IsNotExist(err) {
			err = nil
		} else {
			err = errors.Wrap(err)
		}

		return err
	}

	defer errors.DeferredCloser(&err, file)

	bufferedReader, repool := pool.GetBufferedReader(file)
	defer repool()

	for {
		var object sku.Transacted

		if _, err = coder.DecodeFrom(&object, bufferedReader); err != nil {
			if errors.IsEOF(err) {
				err = nil
			} else {
				err = errors.Wrap(err)
			}

			return err
		}

		if err = each(&object); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}
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

	if store_version.GreaterOrEqual(
		envRepo.GetStoreVersion(),
		store_version.V14,
	) {
		if err = store.flushMutableConfigStreamIndex(envRepo); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else {
		if err = store.flushMutableConfigGob(envRepo); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if err = printerHeader("recompiled konfig"); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *store) flushMutableConfigGob(
	envRepo env_repo.Env,
) (err error) {
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

	return err
}

func (store *store) flushMutableConfigStreamIndex(
	envRepo env_repo.Env,
) (err error) {
	var coder stream_index.ListCoder

	if err = store.flushStreamIndexSingle(
		envRepo.FileConfig(),
		&coder,
		&store.config.Sku,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = store.flushStreamIndexSet(
		envRepo.FileConfigTags(),
		&coder,
		func(each func(*sku.Transacted) error) error {
			for tagObject := range store.config.Tags.All() {
				if err := each(&tagObject.Transacted); err != nil {
					return err
				}
			}
			return nil
		},
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = store.flushStreamIndexSet(
		envRepo.FileConfigTypes(),
		&coder,
		func(each func(*sku.Transacted) error) error {
			for object := range store.config.Types.All() {
				if err := each(object); err != nil {
					return err
				}
			}
			return nil
		},
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = store.flushStreamIndexSet(
		envRepo.FileConfigRepos(),
		&coder,
		func(each func(*sku.Transacted) error) error {
			for object := range store.config.Repos.All() {
				if err := each(object); err != nil {
					return err
				}
			}
			return nil
		},
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *store) flushStreamIndexSingle(
	path string,
	coder *stream_index.ListCoder,
	object *sku.Transacted,
) (err error) {
	var file *os.File

	if file, err = files.OpenCreateWriteOnlyTruncate(path); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, file)

	bufferedWriter, repool := pool.GetBufferedWriter(file)
	defer repool()

	if _, err = coder.EncodeTo(object, bufferedWriter); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = bufferedWriter.Flush(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (store *store) flushStreamIndexSet(
	path string,
	coder *stream_index.ListCoder,
	iter func(func(*sku.Transacted) error) error,
) (err error) {
	var file *os.File

	if file, err = files.OpenCreateWriteOnlyTruncate(path); err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, file)

	bufferedWriter, repool := pool.GetBufferedWriter(file)
	defer repool()

	if err = iter(func(object *sku.Transacted) error {
		if _, err := coder.EncodeTo(object, bufferedWriter); err != nil {
			return errors.Wrap(err)
		}
		return nil
	}); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = bufferedWriter.Flush(); err != nil {
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
