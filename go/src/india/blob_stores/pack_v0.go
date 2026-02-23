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
	digest []byte
	data   []byte
}

type packedBlobMeta struct {
	digest []byte
	size   uint64
}

// splitBlobChunks partitions sorted blob metadata into chunks where each
// chunk's total data size does not exceed maxPackSize. A maxPackSize of 0
// means unlimited (all blobs in one chunk). A single blob larger than the
// limit gets its own chunk.
func splitBlobChunks(metas []packedBlobMeta, maxPackSize uint64) [][]packedBlobMeta {
	if len(metas) == 0 {
		return nil
	}

	if maxPackSize == 0 {
		return [][]packedBlobMeta{metas}
	}

	var chunks [][]packedBlobMeta
	var current []packedBlobMeta
	var currentSize uint64

	for _, meta := range metas {
		if len(current) > 0 && currentSize+meta.size > maxPackSize {
			chunks = append(chunks, current)
			current = nil
			currentSize = 0
		}

		current = append(current, meta)
		currentSize += meta.size
	}

	if len(current) > 0 {
		chunks = append(chunks, current)
	}

	return chunks
}

func (store inventoryArchiveV0) Pack(options PackOptions) (err error) {
	ctx := options.Context
	tw := options.TapWriter

	// Phase 1: Collect metadata only (digest + size). No blob data is
	// retained, bounding memory to the metadata slice.
	var metas []packedBlobMeta

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

		blobSize, sizeErr := store.GetBlobSize(looseId)
		if sizeErr != nil {
			err = errors.Wrapf(sizeErr, "getting size of loose blob %s", looseId)
			tapNotOk(tw, "collect loose blobs", err)
			return err
		}

		digestBytes := make([]byte, len(looseId.GetBytes()))
		copy(digestBytes, looseId.GetBytes())

		metas = append(metas, packedBlobMeta{digest: digestBytes, size: blobSize})
	}

	if len(metas) == 0 {
		return nil
	}

	tapOk(tw, fmt.Sprintf("collect %d loose blobs", len(metas)))

	sort.Slice(metas, func(i, j int) bool {
		return bytes.Compare(metas[i].digest, metas[j].digest) < 0
	})

	maxPackSize := options.MaxPackSize
	if maxPackSize == 0 {
		maxPackSize = store.config.GetMaxPackSize()
	}

	chunks := splitBlobChunks(metas, maxPackSize)
	totalChunks := len(chunks)

	type chunkResult struct {
		dataPath string
		metas    []packedBlobMeta
	}

	var results []chunkResult

	// Phase 2: Load blob data one chunk at a time, write archive, release.
	for chunkIdx, chunkMetas := range chunks {
		if err = packContextCancelled(ctx); err != nil {
			err = errors.Wrap(err)
			return err
		}

		blobs := make([]packedBlob, len(chunkMetas))

		for i, meta := range chunkMetas {
			marklId, repool := store.defaultHash.GetBlobIdForHexString(
				hex.EncodeToString(meta.digest),
			)

			reader, readErr := store.looseBlobStore.MakeBlobReader(marklId)
			repool()

			if readErr != nil {
				err = errors.Wrapf(readErr, "reading loose blob %x", meta.digest)
				tapNotOk(tw, fmt.Sprintf("write chunk %d/%d", chunkIdx+1, totalChunks), err)
				return err
			}

			data, readAllErr := io.ReadAll(reader)
			reader.Close()

			if readAllErr != nil {
				err = errors.Wrapf(readAllErr, "reading loose blob data %x", meta.digest)
				tapNotOk(tw, fmt.Sprintf("write chunk %d/%d", chunkIdx+1, totalChunks), err)
				return err
			}

			blobs[i] = packedBlob{digest: meta.digest, data: data}
		}

		dataPath, entryCount, packErr := store.packChunkArchive(blobs)
		if packErr != nil {
			desc := fmt.Sprintf("write chunk %d/%d", chunkIdx+1, totalChunks)
			tapNotOk(tw, desc, packErr)
			return packErr
		}

		tapOk(tw, fmt.Sprintf(
			"write chunk %d/%d (%d entries, 0 delta)",
			chunkIdx+1, totalChunks, entryCount,
		))

		// Release blob data — let GC reclaim before next chunk.
		blobs = nil

		results = append(results, chunkResult{dataPath: dataPath, metas: chunkMetas})
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

		if err = store.validateArchive(r.dataPath, len(r.metas)); err != nil {
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
			for _, meta := range metas {
				marklId, repool := store.defaultHash.GetBlobIdForHexString(
					hex.EncodeToString(meta.digest),
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

	if err = store.deleteLooseBlobs(ctx, metas); err != nil {
		tapNotOk(tw, fmt.Sprintf("delete %d loose blobs", len(metas)), err)
		return err
	}

	tapOk(tw, fmt.Sprintf("delete %d loose blobs", len(metas)))

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
		if err = dataWriter.WriteEntry(blob.digest, blob.data); err != nil {
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
	expectedCount int,
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

	if len(entries) != expectedCount {
		err = errors.Errorf(
			"archive entry count mismatch: wrote %d, read %d",
			expectedCount,
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
	metas []packedBlobMeta,
) (err error) {
	deleter, ok := store.looseBlobStore.(BlobDeleter)
	if !ok {
		err = errors.Errorf("loose blob store does not support deletion")
		return err
	}

	for _, meta := range metas {
		if err = packContextCancelled(ctx); err != nil {
			err = errors.Wrap(err)
			return err
		}

		marklId, repool := store.defaultHash.GetBlobIdForHexString(
			hex.EncodeToString(meta.digest),
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

func (store inventoryArchiveV0) GetBlobSize(
	id domain_interfaces.MarklId,
) (size uint64, err error) {
	reader, err := store.looseBlobStore.MakeBlobReader(id)
	if err != nil {
		err = errors.Wrapf(err, "opening blob %s for size", id)
		return size, err
	}

	defer errors.DeferredCloser(&err, reader)

	n, err := io.Copy(io.Discard, reader)
	if err != nil {
		err = errors.Wrapf(err, "reading blob %s for size", id)
		return size, err
	}

	return uint64(n), nil
}
