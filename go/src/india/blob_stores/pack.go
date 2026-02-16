package blob_stores

import (
	"bytes"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/echo/inventory_archive"
)

// PackableArchive is implemented by blob stores that support packing loose
// blobs into archive files.
type PackableArchive interface {
	Pack() error
}

func (store inventoryArchive) Pack() (err error) {
	hashFormatId := store.defaultHash.GetMarklFormatId()

	type looseBlob struct {
		hash []byte
		data []byte
	}

	var blobs []looseBlob

	for looseId, iterErr := range store.looseBlobStore.AllBlobs() {
		if iterErr != nil {
			err = errors.Wrap(iterErr)
			return err
		}

		if looseId.IsNull() {
			continue
		}

		if _, inArchive := store.index[looseId.String()]; inArchive {
			continue
		}

		reader, readErr := store.looseBlobStore.MakeBlobReader(looseId)
		if readErr != nil {
			err = errors.Wrapf(readErr, "reading loose blob %s", looseId)
			return err
		}

		data, readAllErr := io.ReadAll(reader)
		reader.Close()

		if readAllErr != nil {
			err = errors.Wrapf(readAllErr, "reading loose blob data %s", looseId)
			return err
		}

		hashBytes := make([]byte, len(looseId.GetBytes()))
		copy(hashBytes, looseId.GetBytes())

		blobs = append(blobs, looseBlob{hash: hashBytes, data: data})
	}

	if len(blobs) == 0 {
		return nil
	}

	sort.Slice(blobs, func(i, j int) bool {
		return bytes.Compare(blobs[i].hash, blobs[j].hash) < 0
	})

	ct := store.config.GetCompressionType()

	var dataBuf bytes.Buffer

	dataWriter, err := inventory_archive.NewDataWriter(
		&dataBuf,
		hashFormatId,
		ct,
	)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	for _, blob := range blobs {
		if err = dataWriter.WriteEntry(blob.hash, blob.data); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	checksum, writtenEntries, err := dataWriter.Close()
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	archiveChecksum := hex.EncodeToString(checksum)

	if err = os.MkdirAll(store.basePath, 0o755); err != nil {
		err = errors.Wrapf(err, "creating archive directory %s", store.basePath)
		return err
	}

	dataPath := filepath.Join(
		store.basePath,
		archiveChecksum+inventory_archive.DataFileExtension,
	)

	if err = os.WriteFile(dataPath, dataBuf.Bytes(), 0o644); err != nil {
		err = errors.Wrapf(err, "writing data file %s", dataPath)
		return err
	}

	// Build and write index file
	indexEntries := make([]inventory_archive.IndexEntry, len(writtenEntries))
	for i, de := range writtenEntries {
		indexEntries[i] = inventory_archive.IndexEntry{
			Hash:           de.Hash,
			PackOffset:     de.Offset,
			CompressedSize: de.CompressedSize,
		}
	}

	var indexBuf bytes.Buffer

	if _, err = inventory_archive.WriteIndex(
		&indexBuf,
		hashFormatId,
		indexEntries,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	indexPath := filepath.Join(
		store.basePath,
		archiveChecksum+inventory_archive.IndexFileExtension,
	)

	if err = os.WriteFile(indexPath, indexBuf.Bytes(), 0o644); err != nil {
		err = errors.Wrapf(err, "writing index file %s", indexPath)
		return err
	}

	// Update in-memory index
	for _, de := range writtenEntries {
		marklId, repool := store.defaultHash.GetBlobIdForHexString(
			hex.EncodeToString(de.Hash),
		)
		key := marklId.String()
		repool()

		store.index[key] = archiveEntry{
			ArchiveChecksum: archiveChecksum,
			Offset:          de.Offset,
			CompressedSize:  de.CompressedSize,
		}
	}

	// Build cache entries from the full in-memory index
	var allCacheEntries []inventory_archive.CacheEntry

	for key, entry := range store.index {
		id, repool := store.defaultHash.GetBlobId()
		if setErr := id.Set(key); setErr != nil {
			repool()
			continue
		}

		hashBytes := make([]byte, len(id.GetBytes()))
		copy(hashBytes, id.GetBytes())
		repool()

		archiveBytes, decodeErr := hex.DecodeString(entry.ArchiveChecksum)
		if decodeErr != nil {
			continue
		}

		allCacheEntries = append(allCacheEntries, inventory_archive.CacheEntry{
			Hash:            hashBytes,
			ArchiveChecksum: archiveBytes,
			Offset:          entry.Offset,
			CompressedSize:  entry.CompressedSize,
		})
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
		inventory_archive.CacheFileName,
	)

	cacheFile, err := os.Create(cachePath)
	if err != nil {
		err = errors.Wrapf(err, "creating cache file %s", cachePath)
		return err
	}

	defer errors.DeferredCloser(&err, cacheFile)

	if _, err = inventory_archive.WriteCache(
		cacheFile,
		hashFormatId,
		allCacheEntries,
	); err != nil {
		err = errors.Wrapf(err, "writing cache file %s", cachePath)
		return err
	}

	return nil
}
