package blob_stores

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl_io"
	"code.linenisgreat.com/dodder/go/internal/bravo/ui"
	"code.linenisgreat.com/dodder/go/internal/echo/inventory_archive"
	"code.linenisgreat.com/dodder/go/internal/golf/blob_store_configs"
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
	ctx := options.Context
	tw := options.TapWriter

	metas, err := collectBlobMetasParallel(
		ctx,
		tw,
		store.looseBlobStore,
		indexPresenceFromV1(store.index),
		options,
		store.GetBlobSize,
	)
	if err != nil {
		return err
	}

	if len(metas) == 0 {
		return nil
	}

	// Split into chunks based on max pack size.
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

		var blobs []packedBlob

		for _, meta := range chunkMetas {
			marklId, repool := store.defaultHash.GetBlobIdForHexString(
				hex.EncodeToString(meta.digest),
			)
			idString := marklId.String()

			reader, readErr := store.looseBlobStore.MakeBlobReader(marklId)
			repool()

			if readErr != nil {
				if options.SkipMissingBlobs {
					tapComment(tw, fmt.Sprintf("blob skipped: %s", idString))
					continue
				}

				err = errors.Wrapf(readErr, "reading loose blob %x", meta.digest)
				tapNotOk(tw, fmt.Sprintf("blob skipped: %s", idString), err)
				return err
			}

			data, readAllErr := io.ReadAll(reader)
			reader.Close()

			if readAllErr != nil {
				if options.SkipMissingBlobs {
					tapComment(tw, fmt.Sprintf("blob skipped: %s", idString))
					continue
				}

				err = errors.Wrapf(readAllErr, "reading loose blob data %x", meta.digest)
				tapNotOk(tw, fmt.Sprintf("blob skipped: %s", idString), err)
				return err
			}

			blobs = append(blobs, packedBlob{digest: meta.digest, data: data})
		}

		if len(blobs) == 0 {
			continue
		}

		var rawSize uint64
		for _, blob := range blobs {
			rawSize += uint64(len(blob.data))
		}

		dataPath, fullCount, deltaCount, packErr := store.packChunkArchiveV1(ctx, blobs)
		if packErr != nil {
			desc := fmt.Sprintf("write archive %d/%d", chunkIdx+1, totalChunks)
			tapNotOk(tw, desc, packErr)
			return packErr
		}

		archiveChecksum := strings.TrimSuffix(
			filepath.Base(dataPath),
			inventory_archive.DataFileExtensionV1,
		)

		entryCount := fullCount + deltaCount

		if fi, statErr := os.Stat(dataPath); statErr == nil {
			archiveSize := uint64(fi.Size())

			var compressionPct float64
			if rawSize > 0 {
				compressionPct = float64(archiveSize) / float64(rawSize) * 100
			}

			tapOk(tw, fmt.Sprintf(
				"write archive %d/%d %s (%d entries, %d delta, %s, %.0f%%)",
				chunkIdx+1, totalChunks,
				archiveChecksum,
				entryCount, deltaCount,
				ui.GetHumanBytesString(archiveSize),
				compressionPct,
			))
		} else {
			tapOk(tw, fmt.Sprintf(
				"write archive %d/%d %s (%d entries, %d delta)",
				chunkIdx+1, totalChunks,
				archiveChecksum,
				entryCount, deltaCount,
			))
		}

		// Release blob data — let GC reclaim before next chunk.
		blobs = nil

		results = append(results, chunkResult{dataPath: dataPath, metas: chunkMetas})
	}

	// Write cache from the full in-memory index.
	if err = store.writeCacheV1(); err != nil {
		tapNotOk(tw, "write cache", err)
		return err
	}

	tapOk(tw, "write cache")

	if !options.DeleteLoose {
		return nil
	}

	// Validate all archives, then delete loose blobs.
	for chunkIdx, r := range results {
		if err = packContextCancelled(ctx); err != nil {
			err = errors.Wrap(err)
			return err
		}

		if err = store.validateArchiveV1(r.dataPath, len(r.metas)); err != nil {
			desc := fmt.Sprintf("validate archive %d/%d", chunkIdx+1, totalChunks)
			tapNotOk(tw, desc, err)
			return err
		}

		tapOk(tw, fmt.Sprintf("validate archive %d/%d", chunkIdx+1, totalChunks))
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

	if err = store.deleteLooseBlobsV1(ctx, metas); err != nil {
		tapNotOk(tw, fmt.Sprintf("delete %d loose blobs", len(metas)), err)
		return err
	}

	tapOk(tw, fmt.Sprintf("delete %d loose blobs", len(metas)))

	return nil
}

func (store inventoryArchiveV1) packChunkArchiveV1(
	ctx interfaces.ActiveContext,
	blobs []packedBlob,
) (dataPath string, fullCount int, deltaCount int, err error) {
	hashFormatId := store.defaultHash.GetMarklFormatId()

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
			return dataPath, 0, 0, err
		}

		alg, algErr = inventory_archive.DeltaAlgorithmForByte(algByte)
		if algErr != nil {
			err = errors.Wrap(algErr)
			return dataPath, 0, 0, err
		}

		// Resolve selector from config.
		sigConfig, hasSigConfig := store.config.(blob_store_configs.SignatureConfigImmutable)
		selConfig, hasSelConfig := store.config.(blob_store_configs.SelectorConfigImmutable)

		var selector inventory_archive.BaseSelector

		if hasSelConfig && selConfig.GetSelectorType() != "" && selConfig.GetSelectorType() != "size-based" {
			var selErr error
			selector, selErr = inventory_archive.BaseSelectorForName(
				selConfig.GetSelectorType(),
				inventory_archive.BaseSelectorParams{
					Bands:       selConfig.GetSelectorBands(),
					RowsPerBand: selConfig.GetSelectorRowsPerBand(),
					MinBlobSize: selConfig.GetSelectorMinBlobSize(),
					MaxBlobSize: selConfig.GetSelectorMaxBlobSize(),
				},
			)
			if selErr != nil {
				err = errors.Wrap(selErr)
				return dataPath, 0, 0, err
			}
		}

		if selector == nil {
			selector = &inventory_archive.SizeBasedSelector{
				MinBlobSize: store.config.GetDeltaMinBlobSize(),
				MaxBlobSize: store.config.GetDeltaMaxBlobSize(),
				SizeRatio:   store.config.GetDeltaSizeRatio(),
			}
		}

		// Build BlobSet.
		blobSet := &sliceBlobSet{
			blobs: make([]inventory_archive.BlobMetadata, len(blobs)),
		}

		for i, blob := range blobs {
			marklId, repool := store.defaultHash.GetBlobIdForHexString(
				hex.EncodeToString(blob.digest),
			)
			blobSet.blobs[i] = inventory_archive.BlobMetadata{
				Id:   marklId,
				Size: uint64(len(blob.data)),
			}
			repool()
		}

		// Compute signatures if configured.
		if hasSigConfig && sigConfig.GetSignatureType() != "" {
			sigComputer, sigErr := inventory_archive.SignatureComputerForName(
				sigConfig.GetSignatureType(),
				inventory_archive.SignatureComputerParams{
					SignatureLen: sigConfig.GetSignatureLen(),
					AvgChunkSize: sigConfig.GetAvgChunkSize(),
					MinChunkSize: sigConfig.GetMinChunkSize(),
					MaxChunkSize: sigConfig.GetMaxChunkSize(),
				},
			)
			if sigErr != nil {
				err = errors.Wrap(sigErr)
				return dataPath, 0, 0, err
			}

			if sigComputer != nil {
				for i, blob := range blobs {
					sig, compErr := sigComputer.ComputeSignature(
						bytes.NewReader(blob.data),
					)
					if compErr != nil {
						err = errors.Wrapf(compErr, "computing signature for blob %d", i)
						return dataPath, 0, 0, err
					}

					blobSet.blobs[i].Signature = sig
				}
			}
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

	// Phase 3: Write data file to a temp file, then rename after checksum.
	if err = os.MkdirAll(store.archivesPath(), 0o755); err != nil {
		err = errors.Wrapf(err, "creating archive directory %s", store.archivesPath())
		return dataPath, 0, 0, err
	}

	tmpFile, err := os.CreateTemp(store.archivesPath(), "pack-*.tmp")
	if err != nil {
		err = errors.Wrapf(err, "creating temp file in %s", store.archivesPath())
		return dataPath, 0, 0, err
	}

	tmpPath := tmpFile.Name()

	defer func() {
		if err != nil {
			os.Remove(tmpPath)
		}
	}()

	dataWriter, err := inventory_archive.NewDataWriterV1(
		tmpFile,
		hashFormatId,
		ct,
		flags,
		store.encryption,
	)
	if err != nil {
		tmpFile.Close()
		err = errors.Wrap(err)
		return dataPath, 0, 0, err
	}

	// First pass: write all blobs NOT assigned as deltas (bases + unassigned).
	for i, blob := range blobs {
		if isDelta[i] {
			continue
		}

		if writeErr := dataWriter.WriteFullEntry(blob.digest, blob.data); writeErr != nil {
			tmpFile.Close()
			err = errors.Wrap(writeErr)
			return dataPath, 0, 0, err
		}
	}

	// Second pass: compute deltas in parallel, then write sequentially.
	type indexedAssignment struct {
		resultIdx int
		blobIdx   int
		baseIdx   int
	}

	var orderedAssignments []indexedAssignment
	for blobIdx, baseIdx := range assignments {
		orderedAssignments = append(orderedAssignments, indexedAssignment{
			blobIdx: blobIdx,
			baseIdx: baseIdx,
		})
	}

	// Sort by blobIdx for deterministic archive output. blobs is already
	// digest-sorted, so blobIdx order preserves that ordering.
	sort.Slice(orderedAssignments, func(i, j int) bool {
		return orderedAssignments[i].blobIdx < orderedAssignments[j].blobIdx
	})

	for i := range orderedAssignments {
		orderedAssignments[i].resultIdx = i
	}

	results := make([]deltaResult, len(orderedAssignments))

	numWorkers := runtime.NumCPU()
	if numWorkers > len(orderedAssignments) {
		numWorkers = len(orderedAssignments)
	}

	if len(orderedAssignments) > 0 {
		sem := make(chan struct{}, numWorkers)

		var wg sync.WaitGroup

		for _, assignment := range orderedAssignments {
			wg.Add(1)
			sem <- struct{}{}

			go func(a indexedAssignment) {
				defer wg.Done()
				defer func() { <-sem }()

				if packContextCancelled(ctx) != nil {
					return
				}

				targetBlob := blobs[a.blobIdx]
				baseBlob := blobs[a.baseIdx]

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
					// Delta computation failed — store as full.
					results[a.resultIdx] = deltaResult{
						blobIdx: a.blobIdx,
						baseIdx: a.baseIdx,
					}
					return
				}

				rawDelta := deltaBuf.Bytes()

				// Trial-and-discard: if delta is not smaller, store as full.
				if len(rawDelta) >= len(targetBlob.data) {
					results[a.resultIdx] = deltaResult{
						blobIdx: a.blobIdx,
						baseIdx: a.baseIdx,
					}
					return
				}

				results[a.resultIdx] = deltaResult{
					blobIdx:   a.blobIdx,
					baseIdx:   a.baseIdx,
					deltaData: rawDelta,
				}
			}(assignment)
		}

		wg.Wait()
	}

	if err = packContextCancelled(ctx); err != nil {
		tmpFile.Close()
		err = errors.Wrap(err)
		return dataPath, 0, 0, err
	}

	// Sequential write pass: write deltas (or full fallbacks) in order.
	for _, dr := range results {
		if err = packContextCancelled(ctx); err != nil {
			tmpFile.Close()
			err = errors.Wrap(err)
			return dataPath, 0, 0, err
		}

		targetBlob := blobs[dr.blobIdx]
		baseBlob := blobs[dr.baseIdx]

		if dr.deltaData == nil {
			// Store as full entry (delta failed or was larger).
			if writeErr := dataWriter.WriteFullEntry(
				targetBlob.digest,
				targetBlob.data,
			); writeErr != nil {
				tmpFile.Close()
				err = errors.Wrap(writeErr)
				return dataPath, 0, 0, err
			}

			continue
		}

		if writeErr := dataWriter.WriteDeltaEntry(
			targetBlob.digest,
			algByte,
			baseBlob.digest,
			uint64(len(targetBlob.data)),
			dr.deltaData,
		); writeErr != nil {
			tmpFile.Close()
			err = errors.Wrap(writeErr)
			return dataPath, 0, 0, err
		}
	}

	checksum, writtenEntries, err := dataWriter.Close()
	if err != nil {
		tmpFile.Close()
		err = errors.Wrap(err)
		return dataPath, 0, 0, err
	}

	if err = tmpFile.Close(); err != nil {
		err = errors.Wrapf(err, "closing temp data file %s", tmpPath)
		return dataPath, 0, 0, err
	}

	archiveChecksum := hex.EncodeToString(checksum)

	dataPath = filepath.Join(
		store.archivesPath(),
		archiveChecksum+inventory_archive.DataFileExtensionV1,
	)

	if err = os.Rename(tmpPath, dataPath); err != nil {
		err = errors.Wrapf(err, "renaming temp data file to %s", dataPath)
		return dataPath, 0, 0, err
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
			Hash:       de.Hash,
			PackOffset: de.Offset,
			StoredSize: de.StoredSize,
			EntryType:  de.EntryType,
			BaseOffset: baseOffset,
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
		return dataPath, 0, 0, err
	}

	indexPath := filepath.Join(
		store.archivesPath(),
		archiveChecksum+inventory_archive.IndexFileExtensionV1,
	)

	if err = os.WriteFile(indexPath, indexBuf.Bytes(), 0o644); err != nil {
		err = errors.Wrapf(err, "writing v1 index file %s", indexPath)
		return dataPath, 0, 0, err
	}

	// Phase 5: Update in-memory index and count entry types.
	for _, de := range writtenEntries {
		if de.EntryType == inventory_archive.EntryTypeDelta {
			deltaCount++
		} else {
			fullCount++
		}

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
			StoredSize:      de.StoredSize,
			EntryType:       de.EntryType,
			BaseOffset:      baseOffset,
		}
	}

	return dataPath, fullCount, deltaCount, nil
}

func (store inventoryArchiveV1) writeCacheV1() (err error) {
	hashFormatId := store.defaultHash.GetMarklFormatId()

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
			StoredSize:      entry.StoredSize,
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

	return nil
}

func (store inventoryArchiveV1) validateArchiveV1(
	dataPath string,
	expectedCount int,
) (err error) {
	file, err := os.Open(dataPath)
	if err != nil {
		err = errors.Wrapf(err, "reopening v1 archive for validation %s", dataPath)
		return err
	}

	defer errors.DeferredCloser(&err, file)

	dataReader, err := inventory_archive.NewDataReaderV1(file, store.encryption)
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

	if len(entries) != expectedCount {
		err = errors.Errorf(
			"v1 archive entry count mismatch: wrote %d, read %d",
			expectedCount,
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

// TODO(near-future): Replace read-and-discard with os.Stat for local
// filesystem stores without compression/encryption. See BlobSizer
// interface comment in pack_parallel.go.
func (store inventoryArchiveV1) GetBlobSize(
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
