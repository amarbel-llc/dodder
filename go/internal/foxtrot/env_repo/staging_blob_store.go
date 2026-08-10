package env_repo

import (
	"os"
	"path/filepath"

	"code.linenisgreat.com/madder/go/pkgs/blob_store_configs"
	"code.linenisgreat.com/madder/go/pkgs/blob_stores"
	mad_directory_layout "code.linenisgreat.com/madder/go/pkgs/directory_layout"
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

// MakeDiscardableStagingBlobStore constructs a content-addressed blob store
// rooted at a fresh run-stamped directory under the repo's XDG cache tree. It
// exists so `transform -dry_run` can honor blobs.write (transform scripts
// assume read-your-writes) WITHOUT ever touching the repo's real blob store:
// the returned store is standalone and never registered, and its directory is
// ALWAYS SAFE TO DELETE — it lives under the cache tree precisely so a
// cache-wipe (or the user) can reclaim it, and nothing in the repo ever
// references it. It is deliberately NOT placed under GetTempLocal(), whose tree
// is removed at process exit and so could not be inspected after a dry run.
//
// The digests it reports are byte-identical to a real write because it reuses
// the repo's LOCAL write-store config (hence its hash type). Under the
// write_through multi default (FDR-0016) the default store's config is a
// ConfigMulti; reusing that directly would send MakeBlobStore down its
// member-resolving branch instead of creating a local store at our temp path,
// so we resolve the multi's local write-store member and reuse THAT config. A
// legacy repo whose default store is already a local config is used directly.
// If no local config is reachable, that is a surfaced error, never a silent
// fallback that might send staged writes to real storage.
//
// The caller owns the returned directory; there is no cleanup here.
func (env Env) MakeDiscardableStagingBlobStore() (
	store blob_stores.BlobStoreInitialized,
	dir string,
	err error,
) {
	var local blob_stores.BlobStoreInitialized

	if local, err = env.resolveLocalWriteStore(); err != nil {
		return store, dir, err
	}

	parent := env.DirCacheRepo("transform-dry_run")

	if err = env.MakeDirs(parent); err != nil {
		err = errors.Wrap(err)
		return store, dir, err
	}

	if dir, err = os.MkdirTemp(parent, "run-*"); err != nil {
		err = errors.Wrapf(err, "creating dry-run staging dir under %q", parent)
		return store, dir, err
	}

	store.Config = local.Config
	store.Path = mad_directory_layout.MakeBlobStorePath(
		local.Path.GetId(),
		dir,
		filepath.Join(dir, "blob_store-config"),
	)

	if store.BlobStore, err = blob_stores.MakeBlobStore(
		env.GetEnvBlobStore(),
		store.ConfigNamed,
		nil,
	); err != nil {
		err = errors.Wrapf(err, "constructing staging blob store at %q", dir)
		return store, dir, err
	}

	return store, dir, err
}

// MakeReadBlobStoreWithOverlay returns a read view that consults the given
// overlays first, in order, and then the repo's normal read stores (default +
// fallbacks). At least one overlay must be given.
//
// It is the read half of the -dry_run containment (dodder#390): pointing both
// the blobs.read FFI and the transform's output validation at this view lets a
// dry run read back blobs its blobs.write calls only staged — so read-your-
// writes holds and validation of an object whose Blob was rewritten to a staged
// digest passes — without an overlay store ever being written to by the real
// read path. dodder#392's init-from-lists also passes its read-only
// -blob-source stores here (ahead of the staging store under -dry_run), so a
// consolidation resolves source blobs it never copied into the newborn.
//
// It mirrors makeReadBlobStore's multi construction, with the first overlay as
// the (never-read-past) write target so the overlays are tried first.
func (env Env) MakeReadBlobStoreWithOverlay(
	overlays ...blob_stores.BlobStoreInitialized,
) mad_domain_interfaces.BlobStore {
	defaultStore := env.blobStoreEnv.GetDefaultBlobStore()
	defaultId := defaultStore.Path.GetId().String()

	overlayIds := make(map[string]struct{}, len(overlays))
	for _, overlay := range overlays {
		overlayIds[overlay.Path.GetId().String()] = struct{}{}
	}

	// The multi reads its write target first, then its Read stores in order:
	// [overlays[0], overlays[1:]…, default, remaining fallbacks].
	readStores := make(
		[]blob_stores.BlobStoreInitialized,
		0,
		len(overlays)+1,
	)
	readStores = append(readStores, overlays[1:]...)
	readStores = append(readStores, defaultStore)

	for _, store := range env.blobStoreEnv.GetBlobStoresSorted() {
		id := store.Path.GetId().String()
		if id == defaultId {
			continue
		}
		if _, isOverlay := overlayIds[id]; isOverlay {
			continue
		}

		readStores = append(readStores, store)
	}

	multi, err := blob_stores.
		NewMulti(env.GetActiveContext()).
		WriteTo(overlays[0]).
		Read(readStores...).
		ReadFill(false).
		Build()
	if err != nil {
		env.Cancel(errors.Wrap(err))
	}

	return multi
}

// resolveLocalWriteStore returns the local content-addressed store the repo
// writes blobs into: the write-store member of the default write_through multi
// (the FDR-0016 default), or the default store itself when it is already a
// local store (legacy). Anything else — a multi with no write store, or a
// default that resolves to a non-local store — is an error rather than a
// silent fallback, so a dry run can never quietly stage into real storage.
func (env Env) resolveLocalWriteStore() (
	blob_stores.BlobStoreInitialized,
	error,
) {
	defaultStore := env.GetDefaultBlobStore()

	if multi, ok := defaultStore.Config.Blob.(blob_store_configs.ConfigMulti); ok {
		writeStoreId := multi.GetWriteStore()

		if writeStoreId.IsEmpty() {
			return defaultStore, errors.ErrorWithStackf(
				"default multi blob store has no write store; cannot stage dry-run blobs",
			)
		}

		writeStore := env.GetEnvBlobStore().GetBlobStore(writeStoreId)

		if _, ok := writeStore.Config.Blob.(blob_store_configs.ConfigLocalHashBucketed); !ok {
			return writeStore, errors.ErrorWithStackf(
				"default multi's write store %q is not a local store (%T); cannot stage dry-run blobs",
				writeStoreId,
				writeStore.Config.Blob,
			)
		}

		return writeStore, nil
	}

	if _, ok := defaultStore.Config.Blob.(blob_store_configs.ConfigLocalHashBucketed); ok {
		return defaultStore, nil
	}

	return defaultStore, errors.ErrorWithStackf(
		"repo has no local blob store to base a dry-run staging store on (default is %T)",
		defaultStore.Config.Blob,
	)
}
