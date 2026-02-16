package blob_stores

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/golf/blob_store_configs"
	"code.linenisgreat.com/dodder/go/src/hotel/env_dir"
)

type archiveEntry struct {
	ArchiveChecksum string // hex filename stem
	Offset          uint64
	CompressedSize  uint64
}

type inventoryArchive struct {
	config         blob_store_configs.ConfigInventoryArchive
	defaultHash    markl.FormatHash
	basePath       string
	cachePath      string
	looseBlobStore domain_interfaces.BlobStore
	index          map[string]archiveEntry // keyed by hex hash
}

var _ domain_interfaces.BlobStore = inventoryArchive{}

func makeInventoryArchive(
	envDir env_dir.Env,
	basePath string,
	config blob_store_configs.ConfigInventoryArchive,
	looseBlobStore domain_interfaces.BlobStore,
) (store inventoryArchive, err error) {
	store.config = config
	store.looseBlobStore = looseBlobStore
	store.basePath = basePath

	if store.defaultHash, err = markl.GetFormatHashOrError(
		config.GetDefaultHashTypeId(),
	); err != nil {
		err = errors.Wrap(err)
		return store, err
	}

	// TODO: derive cache path from envDir XDG cache dir
	// TODO: load index from cache or rebuild

	store.index = make(map[string]archiveEntry)

	return store, err
}

func (store inventoryArchive) GetBlobStoreDescription() string {
	return "local inventory archive"
}

func (store inventoryArchive) GetBlobIOWrapper() domain_interfaces.BlobIOWrapper {
	return store.config
}

func (store inventoryArchive) GetDefaultHashType() domain_interfaces.FormatHash {
	return store.defaultHash
}

func (store inventoryArchive) HasBlob(
	id domain_interfaces.MarklId,
) (ok bool) {
	if id.IsNull() {
		ok = true
		return ok
	}

	if _, ok = store.index[id.String()]; ok {
		return ok
	}

	return store.looseBlobStore.HasBlob(id)
}

func (store inventoryArchive) MakeBlobWriter(
	hashFormat domain_interfaces.FormatHash,
) (blobWriter domain_interfaces.BlobWriter, err error) {
	return store.looseBlobStore.MakeBlobWriter(hashFormat)
}

func (store inventoryArchive) MakeBlobReader(
	id domain_interfaces.MarklId,
) (readCloser domain_interfaces.BlobReader, err error) {
	// TODO: archive reading comes in Task 8
	return store.looseBlobStore.MakeBlobReader(id)
}

func (store inventoryArchive) AllBlobs() interfaces.SeqError[domain_interfaces.MarklId] {
	// TODO: dedup comes in Task 10
	return store.looseBlobStore.AllBlobs()
}
