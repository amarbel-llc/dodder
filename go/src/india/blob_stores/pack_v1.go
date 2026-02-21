package blob_stores

import (
	"bytes"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/markl_io"
	"code.linenisgreat.com/dodder/go/src/echo/inventory_archive"
)

// sliceBlobSet implements inventory_archive.BlobSet backed by a slice.
type sliceBlobSet struct {
	blobs []inventory_archive.BlobMetadata
}

func (s *sliceBlobSet) Len() int {
	return len(s.blobs)
}

func (s *sliceBlobSet) At(index int) inventory_archive.BlobMetadata {
	return s.blobs[index]
}

// mapDeltaAssignments implements inventory_archive.DeltaAssignments using a
// map from blob index to base index.
type mapDeltaAssignments struct {
	assignments map[int]int
}

func (m *mapDeltaAssignments) Assign(blobIndex, baseIndex int) {
	m.assignments[blobIndex] = baseIndex
}

func (m *mapDeltaAssignments) AssignError(blobIndex int, err error) {
	// Errors are treated as non-fatal: the blob will be stored as a full entry.
}

func (store inventoryArchiveV1) Pack(options PackOptions) (err error) {
	// TODO: Collect all blob failures during packing, present summary
	// to user with interactive choices (retry individual, skip to full
	// entry, abort). For now, hard fail on first error.

	hashFormatId := store.defaultHash.GetMarklFormatId()

	// Phase 1: Collect loose blobs not yet in the archive.
	var blobs []packedBlob

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

		blobs = append(blobs, packedBlob{hash: hashBytes, data: data})
	}

	if len(blobs) == 0 {
		return nil
	}

	sort.Slice(blobs, func(i, j int) bool {
		return bytes.Compare(blobs[i].hash, blobs[j].hash) < 0
	})

	// Phase 2: Select delta bases if delta is enabled.
	deltaEnabled := store.config.GetDeltaEnabled()

	// assignments maps blob index -> base index (for deltas)
	assignments := make(map[int]int)

	var algByte byte
	var alg inventory_archive.DeltaAlgorithm

	if deltaEnabled {
		var algErr error

		algByte, algErr = inventory_archive.DeltaAlgorithmByteForName(
			store.config.GetDeltaAlgorithm(),
		)
		if algErr != nil {
			err = errors.Wrap(algErr)
			return err
		}

		alg, algErr = inventory_archive.DeltaAlgorithmForByte(algByte)
		if algErr != nil {
			err = errors.Wrap(algErr)
			return err
		}

		blobSet := &sliceBlobSet{
			blobs: make([]inventory_archive.BlobMetadata, len(blobs)),
		}

		for i, blob := range blobs {
			marklId, repool := store.defaultHash.GetBlobIdForHexString(
				hex.EncodeToString(blob.hash),
			)
			blobSet.blobs[i] = inventory_archive.BlobMetadata{
				Id:   marklId,
				Size: uint64(len(blob.data)),
			}
			repool()
		}

		selector := &inventory_archive.SizeBasedSelector{
			MinBlobSize: store.config.GetDeltaMinBlobSize(),
			MaxBlobSize: store.config.GetDeltaMaxBlobSize(),
			SizeRatio:   store.config.GetDeltaSizeRatio(),
		}

		da := &mapDeltaAssignments{assignments: assignments}
		selector.SelectBases(blobSet, da)
	}

	// Build a set of blob indices assigned as deltas.
	isDelta := make(map[int]bool, len(assignments))
	for blobIdx := range assignments {
		isDelta[blobIdx] = true
	}

	ct := store.config.GetCompressionType()

	// The hasDeltas flag will be set based on whether any deltas were
	// actually written (some may fall back to full during trial-and-discard).
	// We start with FlagHasDeltas if there are assignments, and correct after
	// writing. Actually, since the flag is written in the header before entries,
	// we set it optimistically if assignments exist. If all deltas fall back,
	// the flag is still safe (readers handle archives with the flag set but no
	// actual delta entries).
	var flags uint16
	if len(assignments) > 0 {
		flags = inventory_archive.FlagHasDeltas
	}

	// Phase 3: Write data file.
	var dataBuf bytes.Buffer

	dataWriter, err := inventory_archive.NewDataWriterV1(
		&dataBuf,
		hashFormatId,
		ct,
		flags,
	)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	// First pass: write all blobs NOT assigned as deltas (bases + unassigned).
	for i, blob := range blobs {
		if isDelta[i] {
			continue
		}

		if err = dataWriter.WriteFullEntry(blob.hash, blob.data); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	// Second pass: write delta entries.
	for blobIdx, baseIdx := range assignments {
		targetBlob := blobs[blobIdx]
		baseBlob := blobs[baseIdx]

		baseHash, _ := store.defaultHash.Get() //repool:owned
		baseReader := markl_io.MakeReadCloser(
			baseHash,
			bytes.NewReader(baseBlob.data),
		)

		var deltaBuf bytes.Buffer

		computeErr := alg.Compute(
			baseReader,
			int64(len(baseBlob.data)),
			bytes.NewReader(targetBlob.data),
			&deltaBuf,
		)
		if computeErr != nil {
			// If delta computation fails, write as full entry.
			if err = dataWriter.WriteFullEntry(
				targetBlob.hash,
				targetBlob.data,
			); err != nil {
				err = errors.Wrap(err)
				return err
			}

			continue
		}

		rawDelta := deltaBuf.Bytes()

		// Trial-and-discard: if the raw delta is not smaller than the
		// original data, write as a full entry instead.
		if len(rawDelta) >= len(targetBlob.data) {
			if err = dataWriter.WriteFullEntry(
				targetBlob.hash,
				targetBlob.data,
			); err != nil {
				err = errors.Wrap(err)
				return err
			}

			continue
		}

		if err = dataWriter.WriteDeltaEntry(
			targetBlob.hash,
			algByte,
			baseBlob.hash,
			uint64(len(targetBlob.data)),
			rawDelta,
		); err != nil {
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
		archiveChecksum+inventory_archive.DataFileExtensionV1,
	)

	if err = os.WriteFile(dataPath, dataBuf.Bytes(), 0o644); err != nil {
		err = errors.Wrapf(err, "writing v1 data file %s", dataPath)
		return err
	}

	// Phase 4: Build and write index file.
	// Build a map from hash hex -> offset in the data file for resolving
	// base offsets in delta index entries.
	hashHexToDataOffset := make(map[string]uint64, len(writtenEntries))
	for _, de := range writtenEntries {
		hashHexToDataOffset[hex.EncodeToString(de.Hash)] = de.Offset
	}

	indexEntries := make([]inventory_archive.IndexEntryV1, len(writtenEntries))
	for i, de := range writtenEntries {
		var baseOffset uint64

		if de.EntryType == inventory_archive.EntryTypeDelta {
			baseHashHex := hex.EncodeToString(de.BaseHash)
			baseOffset = hashHexToDataOffset[baseHashHex]
		}

		indexEntries[i] = inventory_archive.IndexEntryV1{
			Hash:           de.Hash,
			PackOffset:     de.Offset,
			CompressedSize: de.CompressedSize,
			EntryType:      de.EntryType,
			BaseOffset:     baseOffset,
		}
	}

	// Sort index entries by hash for the fan-out table.
	sort.Slice(indexEntries, func(i, j int) bool {
		return bytes.Compare(indexEntries[i].Hash, indexEntries[j].Hash) < 0
	})

	var indexBuf bytes.Buffer

	if _, err = inventory_archive.WriteIndexV1(
		&indexBuf,
		hashFormatId,
		indexEntries,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	indexPath := filepath.Join(
		store.basePath,
		archiveChecksum+inventory_archive.IndexFileExtensionV1,
	)

	if err = os.WriteFile(indexPath, indexBuf.Bytes(), 0o644); err != nil {
		err = errors.Wrapf(err, "writing v1 index file %s", indexPath)
		return err
	}

	// Phase 5: Update in-memory index.
	for _, de := range writtenEntries {
		marklId, repool := store.defaultHash.GetBlobIdForHexString(
			hex.EncodeToString(de.Hash),
		)
		key := marklId.String()
		repool()

		var baseOffset uint64
		if de.EntryType == inventory_archive.EntryTypeDelta {
			baseHashHex := hex.EncodeToString(de.BaseHash)
			baseOffset = hashHexToDataOffset[baseHashHex]
		}

		store.index[key] = archiveEntryV1{
			ArchiveChecksum: archiveChecksum,
			Offset:          de.Offset,
			CompressedSize:  de.CompressedSize,
			EntryType:       de.EntryType,
			BaseOffset:      baseOffset,
		}
	}

	// Phase 6: Build and write cache file from the full in-memory index.
	var allCacheEntries []inventory_archive.CacheEntryV1

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

		allCacheEntries = append(allCacheEntries, inventory_archive.CacheEntryV1{
			Hash:            hashBytes,
			ArchiveChecksum: archiveBytes,
			Offset:          entry.Offset,
			CompressedSize:  entry.CompressedSize,
			EntryType:       entry.EntryType,
			BaseOffset:      entry.BaseOffset,
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

	if !options.DeleteLoose {
		return nil
	}

	// Phase 7: Validate archive then delete loose blobs.
	if err = store.validateArchiveV1(dataPath, blobs); err != nil {
		return err
	}

	// Check deletion precondition if one was provided.
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

	if err = store.deleteLooseBlobsV1(blobs); err != nil {
		return err
	}

	return nil
}

func (store inventoryArchiveV1) validateArchiveV1(
	dataPath string,
	blobs []packedBlob,
) (err error) {
	file, err := os.Open(dataPath)
	if err != nil {
		err = errors.Wrapf(err, "reopening v1 archive for validation %s", dataPath)
		return err
	}

	defer errors.DeferredCloser(&err, file)

	dataReader, err := inventory_archive.NewDataReaderV1(file)
	if err != nil {
		err = errors.Wrapf(
			err,
			"reading v1 archive header for validation %s",
			dataPath,
		)
		return err
	}

	entries, err := dataReader.ReadAllEntries()
	if err != nil {
		err = errors.Wrapf(
			err,
			"reading v1 archive entries for validation %s",
			dataPath,
		)
		return err
	}

	if len(entries) != len(blobs) {
		err = errors.Errorf(
			"v1 archive entry count mismatch: wrote %d, read %d",
			len(blobs),
			len(entries),
		)
		return err
	}

	// Build a map of base data by hash for delta reconstruction during
	// validation.
	baseDataByHash := make(map[string][]byte)
	for _, entry := range entries {
		if entry.EntryType == inventory_archive.EntryTypeFull {
			baseDataByHash[hex.EncodeToString(entry.Hash)] = entry.Data
		}
	}

	for i, entry := range entries {
		var originalData []byte

		if entry.EntryType == inventory_archive.EntryTypeFull {
			originalData = entry.Data
		} else {
			// Delta: reconstruct
			baseHashHex := hex.EncodeToString(entry.BaseHash)
			baseData, ok := baseDataByHash[baseHashHex]

			if !ok {
				err = errors.Errorf(
					"v1 archive validation: delta entry %d references "+
						"unknown base %s",
					i,
					baseHashHex,
				)
				return err
			}

			deltaAlg, algErr := inventory_archive.DeltaAlgorithmForByte(
				entry.DeltaAlgorithm,
			)
			if algErr != nil {
				err = errors.Wrapf(algErr, "validation: entry %d", i)
				return err
			}

			baseHash, _ := store.defaultHash.Get() //repool:owned
			baseReader := markl_io.MakeReadCloser(
				baseHash,
				bytes.NewReader(baseData),
			)

			var reconstructedBuf bytes.Buffer

			if applyErr := deltaAlg.Apply(
				baseReader,
				int64(len(baseData)),
				bytes.NewReader(entry.Data),
				&reconstructedBuf,
			); applyErr != nil {
				err = errors.Wrapf(
					applyErr,
					"validation: applying delta for entry %d",
					i,
				)
				return err
			}

			originalData = reconstructedBuf.Bytes()
		}

		hash, hashRepool := store.defaultHash.Get()
		hash.Write(originalData)
		computed := hash.Sum(nil)
		hashRepool()

		if !bytes.Equal(computed, entry.Hash) {
			err = errors.Errorf(
				"v1 archive validation failed: entry %d hash mismatch "+
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

func (store inventoryArchiveV1) deleteLooseBlobsV1(
	blobs []packedBlob,
) (err error) {
	deleter, ok := store.looseBlobStore.(BlobDeleter)
	if !ok {
		err = errors.Errorf("loose blob store does not support deletion")
		return err
	}

	for _, blob := range blobs {
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
