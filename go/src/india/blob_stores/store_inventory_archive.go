package blob_stores

import (
	"bytes"
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/markl_io"
	"code.linenisgreat.com/dodder/go/src/bravo/ohio"
	"code.linenisgreat.com/dodder/go/src/echo/inventory_archive"
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
	if id.IsNull() {
		readCloser = markl_io.MakeNopReadCloser(
			store.defaultHash.Get(),
			ohio.NopCloser(bytes.NewReader(nil)),
		)
		return readCloser, err
	}

	entry, inArchive := store.index[id.String()]
	if !inArchive {
		return store.looseBlobStore.MakeBlobReader(id)
	}

	archivePath := filepath.Join(
		store.basePath,
		entry.ArchiveChecksum+inventory_archive.DataFileExtension,
	)

	file, err := os.Open(archivePath)
	if err != nil {
		err = errors.Wrapf(err, "opening archive %s", archivePath)
		return readCloser, err
	}

	defer errors.DeferredCloser(&err, file)

	dataReader, err := inventory_archive.NewDataReader(file)
	if err != nil {
		err = errors.Wrapf(err, "reading archive header %s", archivePath)
		return readCloser, err
	}

	dataEntry, err := dataReader.ReadEntryAt(entry.Offset)
	if err != nil {
		err = errors.Wrapf(
			err,
			"reading entry at offset %d in %s",
			entry.Offset,
			archivePath,
		)
		return readCloser, err
	}

	hash := store.defaultHash.Get()

	readCloser = markl_io.MakeReadCloser(
		hash,
		bytes.NewReader(dataEntry.Data),
	)

	return readCloser, err
}

func (store inventoryArchive) AllBlobs() interfaces.SeqError[domain_interfaces.MarklId] {
	// TODO: dedup comes in Task 10
	return store.looseBlobStore.AllBlobs()
}
