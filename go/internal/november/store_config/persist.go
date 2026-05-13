package store_config

import (
	"os"

	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/type_blobs"
	"code.linenisgreat.com/dodder/go/internal/hotel/stream_index"
	"code.linenisgreat.com/dodder/go/internal/india/typed_blob_store"
	"code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/alfa/pool"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/values"
	"github.com/amarbel-llc/purse-first/libs/dewey/delta/collections_value"
	"github.com/amarbel-llc/purse-first/libs/dewey/delta/files"
)

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
			if repool != nil {
				repool()
			}

			err = errors.Wrap(err)
			return err
		}

		if commonBlob == nil {
			repool()

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

		repool()
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
	if err = store.loadMutableConfigStreamIndex(envRepo); err != nil {
		err = errors.Wrapf(
			err,
			"failed to load store config (stale format?), try running 'dodder reindex'",
		)
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
			b, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned
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
			b, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned
			sku.Resetter.ResetWith(b, object)
			store.config.Repos.Add(b)
			return nil
		},
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if store.config.Sku.GetType().IsEmpty() {
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

	if err = store.flushMutableConfigStreamIndex(envRepo); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = printerHeader("recompiled konfig"); err != nil {
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
	blobId mad_domain_interfaces.MarklId,
) (err error) {
	// Try every enumerated madder blob store, default first, then
	// remaining (matches madder.cat's fallback pattern). Dodder's mutable
	// config blob is content-addressed: any store holding the blob is a
	// valid source. Pinning the lookup to GetDefaultBlobStore() split-brains
	// in multi-repo flows (clone/pull/push), where the local Repo's
	// .default and a remote's .default may resolve to different
	// directories in the same dodder process. See
	// https://github.com/amarbel-llc/dodder/issues/196.
	blobReader, err := store.openBlobReaderAcrossStores(blobId)
	if err != nil {
		ui.Debug().PrintDebug(store.envRepo.GetXDG())
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, blobReader)

	typedBlob := repo_configs.TypedBlob{
		Type: mutableConfigType.ToMadder(),
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

// openBlobReaderAcrossStores searches every enumerated madder blob store
// for a content-addressed blob. Tries the default store first, then any
// remaining stores returned by GetDefaultBlobStoreAndRemaining. Returns
// the first reader that opens cleanly; surfaces the default store's
// error only if every store misses, so the error message keeps pointing
// at the canonical local path (matching the pre-multi-store behavior).
func (store *store) openBlobReaderAcrossStores(
	blobId mad_domain_interfaces.MarklId,
) (mad_domain_interfaces.BlobReader, error) {
	defaultStore, remaining := store.envRepo.GetDefaultBlobStoreAndRemaining()

	reader, defaultErr := defaultStore.MakeBlobReader(blobId)
	if defaultErr == nil {
		return reader, nil
	}

	for _, blobStore := range remaining {
		if !blobStore.HasBlob(blobId) {
			continue
		}

		if r, err := blobStore.MakeBlobReader(blobId); err == nil {
			return r, nil
		}
	}

	return nil, defaultErr
}
