package blob_stores

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

type archiveEntryV1 struct {
	ArchiveChecksum string // hex filename stem
	Offset          uint64
	CompressedSize  uint64
	EntryType       byte
	BaseOffset      uint64
}

type inventoryArchiveV1 struct {
	config         blob_store_configs.ConfigInventoryArchiveDelta
	defaultHash    markl.FormatHash
	basePath       string
	cachePath      string
	looseBlobStore domain_interfaces.BlobStore
	index          map[string]archiveEntryV1 // keyed by hex hash
}

var _ domain_interfaces.BlobStore = inventoryArchiveV1{}

func makeInventoryArchiveV1(
	envDir env_dir.Env,
	basePath string,
	config blob_store_configs.ConfigInventoryArchiveDelta,
	looseBlobStore domain_interfaces.BlobStore,
) (store inventoryArchiveV1, err error) {
	store.config = config
	store.looseBlobStore = looseBlobStore
	store.basePath = basePath

	if store.defaultHash, err = markl.GetFormatHashOrError(
		config.GetDefaultHashTypeId(),
	); err != nil {
		err = errors.Wrap(err)
		return store, err
	}

	store.cachePath = envDir.GetXDGForBlobStores().Cache.MakePath(
		"inventory-archives",
	).String()

	store.index = make(map[string]archiveEntryV1)

	if err = store.loadIndex(); err != nil {
		err = errors.Wrap(err)
		return store, err
	}

	return store, err
}

func (store *inventoryArchiveV1) loadIndex() (err error) {
	entries, ok := store.tryReadCache()
	if !ok {
		return store.rebuildIndex()
	}

	for _, entry := range entries {
		marklId, repool := store.defaultHash.GetBlobIdForHexString(
			hex.EncodeToString(entry.Hash),
		)
		key := marklId.String()
		repool()

		store.index[key] = archiveEntryV1{
			ArchiveChecksum: hex.EncodeToString(entry.ArchiveChecksum),
			Offset:          entry.Offset,
			CompressedSize:  entry.CompressedSize,
			EntryType:       entry.EntryType,
			BaseOffset:      entry.BaseOffset,
		}
	}

	return nil
}

func (store *inventoryArchiveV1) tryReadCache() (
	entries []inventory_archive.CacheEntryV1,
	ok bool,
) {
	cachePath := filepath.Join(store.cachePath, inventory_archive.CacheFileNameV1)

	file, err := os.Open(cachePath)
	if err != nil {
		return nil, false
	}

	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, false
	}

	hashFormatId := store.defaultHash.GetMarklFormatId()

	reader, err := inventory_archive.NewCacheReaderV1(
		file,
		info.Size(),
		hashFormatId,
	)
	if err != nil {
		return nil, false
	}

	entries, err = reader.ReadAllEntries()
	if err != nil {
		return nil, false
	}

	return entries, true
}

func (store *inventoryArchiveV1) rebuildIndex() (err error) {
	pattern := filepath.Join(
		store.basePath,
		"*"+inventory_archive.IndexFileExtensionV1,
	)

	matches, err := filepath.Glob(pattern)
	if err != nil {
		err = errors.Wrapf(err, "globbing v1 index files")
		return err
	}

	hashFormatId := store.defaultHash.GetMarklFormatId()

	var allCacheEntries []inventory_archive.CacheEntryV1

	for _, indexPath := range matches {
		base := filepath.Base(indexPath)
		archiveChecksum := strings.TrimSuffix(
			base,
			inventory_archive.IndexFileExtensionV1,
		)

		archiveChecksumBytes, decodeErr := hex.DecodeString(archiveChecksum)
		if decodeErr != nil {
			continue
		}

		file, openErr := os.Open(indexPath)
		if openErr != nil {
			err = errors.Wrapf(openErr, "opening v1 index %s", indexPath)
			return err
		}

		info, statErr := file.Stat()
		if statErr != nil {
			file.Close()
			err = errors.Wrapf(statErr, "stat v1 index %s", indexPath)
			return err
		}

		reader, readerErr := inventory_archive.NewIndexReaderV1(
			file,
			info.Size(),
			hashFormatId,
		)
		if readerErr != nil {
			file.Close()
			err = errors.Wrapf(readerErr, "reading v1 index %s", indexPath)
			return err
		}

		indexEntries, readErr := reader.ReadAllEntries()

		file.Close()

		if readErr != nil {
			err = errors.Wrapf(readErr, "reading entries from v1 index %s", indexPath)
			return err
		}

		for _, ie := range indexEntries {
			marklId, repool := store.defaultHash.GetBlobIdForHexString(
				hex.EncodeToString(ie.Hash),
			)
			key := marklId.String()
			repool()

			store.index[key] = archiveEntryV1{
				ArchiveChecksum: archiveChecksum,
				Offset:          ie.PackOffset,
				CompressedSize:  ie.CompressedSize,
				EntryType:       ie.EntryType,
				BaseOffset:      ie.BaseOffset,
			}

			allCacheEntries = append(
				allCacheEntries,
				inventory_archive.CacheEntryV1{
					Hash:            ie.Hash,
					ArchiveChecksum: archiveChecksumBytes,
					Offset:          ie.PackOffset,
					CompressedSize:  ie.CompressedSize,
					EntryType:       ie.EntryType,
					BaseOffset:      ie.BaseOffset,
				},
			)
		}
	}

	if len(allCacheEntries) == 0 {
		return nil
	}

	sort.Slice(allCacheEntries, func(i, j int) bool {
		return bytes.Compare(
			allCacheEntries[i].Hash,
			allCacheEntries[j].Hash,
		) < 0
	})

	if err = os.MkdirAll(store.cachePath, 0o755); err != nil {
		err = errors.Wrapf(err, "creating cache directory %s", store.cachePath)
		return err
	}

	cachePath := filepath.Join(
		store.cachePath,
		inventory_archive.CacheFileNameV1,
	)

	cacheFile, err := os.Create(cachePath)
	if err != nil {
		err = errors.Wrapf(err, "creating v1 cache file %s", cachePath)
		return err
	}

	defer errors.DeferredCloser(&err, cacheFile)

	if _, err = inventory_archive.WriteCacheV1(
		cacheFile,
		hashFormatId,
		allCacheEntries,
	); err != nil {
		err = errors.Wrapf(err, "writing v1 cache file %s", cachePath)
		return err
	}

	return nil
}

func (store inventoryArchiveV1) GetBlobStoreDescription() string {
	return "local inventory archive v1"
}

func (store inventoryArchiveV1) GetBlobIOWrapper() domain_interfaces.BlobIOWrapper {
	return store.config
}

func (store inventoryArchiveV1) GetDefaultHashType() domain_interfaces.FormatHash {
	return store.defaultHash
}

func (store inventoryArchiveV1) HasBlob(
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

func (store inventoryArchiveV1) MakeBlobWriter(
	hashFormat domain_interfaces.FormatHash,
) (blobWriter domain_interfaces.BlobWriter, err error) {
	return store.looseBlobStore.MakeBlobWriter(hashFormat)
}

func (store inventoryArchiveV1) MakeBlobReader(
	id domain_interfaces.MarklId,
) (readCloser domain_interfaces.BlobReader, err error) {
	if id.IsNull() {
		hash, _ := store.defaultHash.Get() //repool:owned
		readCloser = markl_io.MakeNopReadCloser(
			hash,
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
		entry.ArchiveChecksum+inventory_archive.DataFileExtensionV1,
	)

	file, err := os.Open(archivePath)
	if err != nil {
		err = errors.Wrapf(err, "opening v1 archive %s", archivePath)
		return readCloser, err
	}

	// Safe to defer-close: ReadEntryAt fully materializes decompressed data
	// into dataEntry.Data before returning, so the file is not needed after.
	defer errors.DeferredCloser(&err, file)

	dataReader, err := inventory_archive.NewDataReaderV1(file)
	if err != nil {
		err = errors.Wrapf(err, "reading v1 archive header %s", archivePath)
		return readCloser, err
	}

	dataEntry, err := dataReader.ReadEntryAt(entry.Offset)
	if err != nil {
		err = errors.Wrapf(
			err,
			"reading v1 entry at offset %d in %s",
			entry.Offset,
			archivePath,
		)
		return readCloser, err
	}

	hash, _ := store.defaultHash.Get() //repool:owned

	if dataEntry.EntryType == inventory_archive.EntryTypeFull {
		readCloser = markl_io.MakeReadCloser(
			hash,
			bytes.NewReader(dataEntry.Data),
		)
		return readCloser, err
	}

	// Delta entry: reconstruct from base + delta
	baseHashHex := hex.EncodeToString(dataEntry.BaseHash)
	baseId, baseRepool := store.defaultHash.GetBlobIdForHexString(baseHashHex)
	baseEntry, baseInArchive := store.index[baseId.String()]
	baseRepool()

	if !baseInArchive {
		err = errors.Errorf(
			"delta entry references base %s which is not in the archive",
			baseHashHex,
		)
		return readCloser, err
	}

	baseDataEntry, err := dataReader.ReadEntryAt(baseEntry.Offset)
	if err != nil {
		err = errors.Wrapf(
			err,
			"reading base entry at offset %d in %s",
			baseEntry.Offset,
			archivePath,
		)
		return readCloser, err
	}

	if baseDataEntry.EntryType != inventory_archive.EntryTypeFull {
		err = errors.Errorf(
			"delta entry base at offset %d is itself a delta (chained deltas not supported)",
			baseEntry.Offset,
		)
		return readCloser, err
	}

	alg, err := inventory_archive.DeltaAlgorithmForByte(dataEntry.DeltaAlgorithm)
	if err != nil {
		err = errors.Wrap(err)
		return readCloser, err
	}

	baseHash, _ := store.defaultHash.Get() //repool:owned
	baseReader := markl_io.MakeReadCloser(
		baseHash,
		bytes.NewReader(baseDataEntry.Data),
	)

	var reconstructedBuf bytes.Buffer

	if err = alg.Apply(
		baseReader,
		int64(len(baseDataEntry.Data)),
		bytes.NewReader(dataEntry.Data),
		&reconstructedBuf,
	); err != nil {
		err = errors.Wrapf(err, "applying delta for %s", id)
		return readCloser, err
	}

	readCloser = markl_io.MakeReadCloser(
		hash,
		bytes.NewReader(reconstructedBuf.Bytes()),
	)

	return readCloser, err
}

func (store inventoryArchiveV1) AllBlobs() interfaces.SeqError[domain_interfaces.MarklId] {
	return func(yield func(domain_interfaces.MarklId, error) bool) {
		id, repool := store.defaultHash.GetBlobId()
		defer repool()

		// Yield all archive index entries first
		for key := range store.index {
			if err := id.Set(key); err != nil {
				if !yield(nil, errors.Wrap(err)) {
					return
				}

				continue
			}

			if !yield(id, nil) {
				return
			}
		}

		// Yield loose blobs, skipping those already in the archive index
		for looseId, err := range store.looseBlobStore.AllBlobs() {
			if err != nil {
				if !yield(nil, err) {
					return
				}

				continue
			}

			if _, inArchive := store.index[looseId.String()]; inArchive {
				continue
			}

			if !yield(looseId, nil) {
				return
			}
		}
	}
}
