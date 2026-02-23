package blob_stores

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/echo/inventory_archive"
)

type packedBlob struct {
	hash []byte
	data []byte
}

// splitBlobChunks partitions sorted blobs into chunks where each chunk's
// total data size does not exceed maxPackSize. A maxPackSize of 0 means
// unlimited (all blobs in one chunk). A single blob larger than the limit
// gets its own chunk.
func splitBlobChunks(blobs []packedBlob, maxPackSize uint64) [][]packedBlob {
	if len(blobs) == 0 {
		return nil
	}

	if maxPackSize == 0 {
		return [][]packedBlob{blobs}
	}

	var chunks [][]packedBlob
	var current []packedBlob
	var currentSize uint64

	for _, blob := range blobs {
		blobSize := uint64(len(blob.data))

		if len(current) > 0 && currentSize+blobSize > maxPackSize {
			chunks = append(chunks, current)
			current = nil
			currentSize = 0
		}

		current = append(current, blob)
		currentSize += blobSize
	}

	if len(current) > 0 {
		chunks = append(chunks, current)
	}

	return chunks
}

func (store inventoryArchiveV0) Pack(options PackOptions) (err error) {
	ctx := options.Context
	tw := options.TapWriter

	// TODO(P2): collect metadata only (hash + size), split into chunks by
	// MaxPackSize, then load one chunk at a time. This eliminates the
	// all-blobs-in-RAM requirement that causes OOM on large stores.
	var blobs []packedBlob

	for looseId, iterErr := range store.looseBlobStore.AllBlobs() {
		if err = packContextCancelled(ctx); err != nil {
			err = errors.Wrap(err)
			tapNotOk(tw, "collect loose blobs", err)
			return err
		}

		if iterErr != nil {
			err = errors.Wrap(iterErr)
			tapNotOk(tw, "collect loose blobs", err)
			return err
		}

		if looseId.IsNull() {
			continue
		}

		if _, inArchive := store.index[looseId.String()]; inArchive {
			continue
		}

		if options.BlobFilter != nil {
			if _, inFilter := options.BlobFilter[looseId.String()]; !inFilter {
				continue
			}
		}

		reader, readErr := store.looseBlobStore.MakeBlobReader(looseId)
		if readErr != nil {
			err = errors.Wrapf(readErr, "reading loose blob %s", looseId)
			tapNotOk(tw, "collect loose blobs", err)
			return err
		}

		data, readAllErr := io.ReadAll(reader)
		reader.Close()

		if readAllErr != nil {
			err = errors.Wrapf(readAllErr, "reading loose blob data %s", looseId)
			tapNotOk(tw, "collect loose blobs", err)
			return err
		}

		hashBytes := make([]byte, len(looseId.GetBytes()))
		copy(hashBytes, looseId.GetBytes())

		blobs = append(blobs, packedBlob{hash: hashBytes, data: data})
	}

	if len(blobs) == 0 {
		return nil
	}

	tapOk(tw, fmt.Sprintf("collect %d loose blobs", len(blobs)))

	sort.Slice(blobs, func(i, j int) bool {
		return bytes.Compare(blobs[i].hash, blobs[j].hash) < 0
	})

	maxPackSize := options.MaxPackSize
	if maxPackSize == 0 {
		maxPackSize = store.config.GetMaxPackSize()
	}

	chunks := splitBlobChunks(blobs, maxPackSize)
	totalChunks := len(chunks)

	type chunkResult struct {
		dataPath string
		blobs    []packedBlob
	}

	var results []chunkResult

	for chunkIdx, chunk := range chunks {
		if err = packContextCancelled(ctx); err != nil {
			err = errors.Wrap(err)
			return err
		}

		dataPath, entryCount, packErr := store.packChunkArchive(chunk)
		if packErr != nil {
			desc := fmt.Sprintf("write chunk %d/%d", chunkIdx+1, totalChunks)
			tapNotOk(tw, desc, packErr)
			return packErr
		}

		tapOk(tw, fmt.Sprintf(
			"write chunk %d/%d (%d entries, 0 delta)",
			chunkIdx+1, totalChunks, entryCount,
		))

		results = append(results, chunkResult{dataPath: dataPath, blobs: chunk})
	}

	if err = store.writeCache(); err != nil {
		tapNotOk(tw, "write cache", err)
		return err
	}

	tapOk(tw, "write cache")

	if !options.DeleteLoose {
		return nil
	}

	for chunkIdx, r := range results {
		if err = packContextCancelled(ctx); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if err = store.validateArchive(r.dataPath, r.blobs); err != nil {
			desc := fmt.Sprintf("validate chunk %d/%d", chunkIdx+1, totalChunks)
			tapNotOk(tw, desc, err)
			return err
		}

		tapOk(tw, fmt.Sprintf("validate chunk %d/%d", chunkIdx+1, totalChunks))
	}

	if options.DeletionPrecondition != nil {
		blobSeq := func(
			yield func(domain_interfaces.MarklId, error) bool,
		) {
			for _, blob := range blobs {
				marklId, repool := store.defaultHash.GetBlobIdForHexString(
					hex.EncodeToString(blob.hash),
				)

				if !yield(marklId, nil) {
					repool()
					return
				}

				repool()
			}
		}

		if err = options.DeletionPrecondition.CheckBlobsSafeToDelete(
			blobSeq,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if err = store.deleteLooseBlobs(ctx, blobs); err != nil {
		tapNotOk(tw, fmt.Sprintf("delete %d loose blobs", len(blobs)), err)
		return err
	}

	tapOk(tw, fmt.Sprintf("delete %d loose blobs", len(blobs)))

	return nil
}

func (store inventoryArchiveV0) packChunkArchive(
	blobs []packedBlob,
) (dataPath string, entryCount int, err error) {
	hashFormatId := store.defaultHash.GetMarklFormatId()
	ct := store.config.GetCompressionType()

	if mkdirErr := os.MkdirAll(store.basePath, 0o755); mkdirErr != nil {
		err = errors.Wrapf(mkdirErr, "creating archive directory %s", store.basePath)
		return dataPath, 0, err
	}

	tmpFile, err := os.CreateTemp(store.basePath, "pack-*.tmp")
	if err != nil {
		err = errors.Wrapf(err, "creating temp file in %s", store.basePath)
		return dataPath, 0, err
	}

	tmpPath := tmpFile.Name()

	defer func() {
		if err != nil {
			os.Remove(tmpPath)
		}
	}()

	dataWriter, err := inventory_archive.NewDataWriter(
		tmpFile,
		hashFormatId,
		ct,
	)
	if err != nil {
		tmpFile.Close()
		err = errors.Wrap(err)
		return dataPath, 0, err
	}

	for _, blob := range blobs {
		if err = dataWriter.WriteEntry(blob.hash, blob.data); err != nil {
			tmpFile.Close()
			err = errors.Wrap(err)
			return dataPath, 0, err
		}
	}

	checksum, writtenEntries, err := dataWriter.Close()
	if err != nil {
		tmpFile.Close()
		err = errors.Wrap(err)
		return dataPath, 0, err
	}

	if err = tmpFile.Close(); err != nil {
		err = errors.Wrapf(err, "closing temp data file %s", tmpPath)
		return dataPath, 0, err
	}

	archiveChecksum := hex.EncodeToString(checksum)

	dataPath = filepath.Join(
		store.basePath,
		archiveChecksum+inventory_archive.DataFileExtension,
	)

	if err = os.Rename(tmpPath, dataPath); err != nil {
		err = errors.Wrapf(err, "renaming temp data file to %s", dataPath)
		return dataPath, 0, err
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
		return dataPath, 0, err
	}

	indexPath := filepath.Join(
		store.basePath,
		archiveChecksum+inventory_archive.IndexFileExtension,
	)

	if err = os.WriteFile(indexPath, indexBuf.Bytes(), 0o644); err != nil {
		err = errors.Wrapf(err, "writing index file %s", indexPath)
		return dataPath, 0, err
	}

	entryCount = len(writtenEntries)

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

	return dataPath, entryCount, nil
}

func (store inventoryArchiveV0) writeCache() (err error) {
	hashFormatId := store.defaultHash.GetMarklFormatId()

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

func (store inventoryArchiveV0) validateArchive(
	dataPath string,
	blobs []packedBlob,
) (err error) {
	file, err := os.Open(dataPath)
	if err != nil {
		err = errors.Wrapf(err, "reopening archive for validation %s", dataPath)
		return err
	}

	defer errors.DeferredCloser(&err, file)

	dataReader, err := inventory_archive.NewDataReader(file)
	if err != nil {
		err = errors.Wrapf(
			err,
			"reading archive header for validation %s",
			dataPath,
		)
		return err
	}

	entries, err := dataReader.ReadAllEntries()
	if err != nil {
		err = errors.Wrapf(
			err,
			"reading archive entries for validation %s",
			dataPath,
		)
		return err
	}

	if len(entries) != len(blobs) {
		err = errors.Errorf(
			"archive entry count mismatch: wrote %d, read %d",
			len(blobs),
			len(entries),
		)
		return err
	}

	for i, entry := range entries {
		hash, hashRepool := store.defaultHash.Get()
		hash.Write(entry.Data)
		computed := hash.Sum(nil)
		hashRepool()

		if !bytes.Equal(computed, entry.Hash) {
			err = errors.Errorf(
				"archive validation failed: entry %d hash mismatch "+
					"(expected %x, got %x)",
				i,
				entry.Hash,
				computed,
			)
			return err
		}
	}

	return nil
}

func (store inventoryArchiveV0) deleteLooseBlobs(
	ctx interfaces.ActiveContext,
	blobs []packedBlob,
) (err error) {
	deleter, ok := store.looseBlobStore.(BlobDeleter)
	if !ok {
		err = errors.Errorf("loose blob store does not support deletion")
		return err
	}

	for _, blob := range blobs {
		if err = packContextCancelled(ctx); err != nil {
			err = errors.Wrap(err)
			return err
		}

		marklId, repool := store.defaultHash.GetBlobIdForHexString(
			hex.EncodeToString(blob.hash),
		)

		if deleteErr := deleter.DeleteBlob(marklId); deleteErr != nil {
			repool()
			err = errors.Wrap(deleteErr)
			return err
		}

		repool()
	}

	return nil
}
