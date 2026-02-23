package blob_stores

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	tap "github.com/amarbel-llc/tap-dancer/go"
)

// blobSizeFn returns the uncompressed size of a blob given its ID.
// Implementations may read through the full decompression pipeline or use
// a fast path like os.Stat when available.
type blobSizeFn func(domain_interfaces.MarklId) (uint64, error)

// collectBlobMetasParallel iterates the loose blob store to find packing
// candidates, then fans out size lookups across multiple goroutines.
//
// The AllBlobs iterator is consumed serially (it is not concurrent-safe).
// Size lookups are parallel with min(NumCPU, len(candidates)) workers.
func collectBlobMetasParallel(
	ctx interfaces.ActiveContext,
	tw *tap.Writer,
	looseBlobStore domain_interfaces.BlobStore,
	index map[string]bool,
	options PackOptions,
	sizeFn blobSizeFn,
) (metas []packedBlobMeta, err error) {
	// Phase 1a: Serial iteration to collect candidate IDs.
	type candidate struct {
		id     domain_interfaces.MarklId
		digest []byte
	}

	var candidates []candidate

	for looseId, iterErr := range looseBlobStore.AllBlobs() {
		if err = packContextCancelled(ctx); err != nil {
			err = errors.Wrap(err)
			tapNotOk(tw, "collect loose blobs", err)
			return nil, err
		}

		if iterErr != nil {
			err = errors.Wrap(iterErr)
			tapNotOk(tw, "collect loose blobs", err)
			return nil, err
		}

		if looseId.IsNull() {
			continue
		}

		if index[looseId.String()] {
			continue
		}

		if options.BlobFilter != nil {
			if _, inFilter := options.BlobFilter[looseId.String()]; !inFilter {
				continue
			}
		}

		digestBytes := make([]byte, len(looseId.GetBytes()))
		copy(digestBytes, looseId.GetBytes())

		candidates = append(candidates, candidate{
			id:     looseId,
			digest: digestBytes,
		})
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	// Phase 1b: Parallel size lookups.
	metas = make([]packedBlobMeta, len(candidates))

	numWorkers := runtime.NumCPU()
	if numWorkers > len(candidates) {
		numWorkers = len(candidates)
	}

	sem := make(chan struct{}, numWorkers)

	var (
		wg       sync.WaitGroup
		firstErr error
		errOnce  sync.Once
		cancel   context.CancelFunc
	)

	sizeCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i, c := range candidates {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, cand candidate) {
			defer wg.Done()
			defer func() { <-sem }()

			// Check both internal cancellation (first error) and external
			// cancellation (caller's context, e.g. signals, memory pressure).
			select {
			case <-sizeCtx.Done():
				return
			default:
			}

			if packContextCancelled(ctx) != nil {
				return
			}

			blobSize, sizeErr := sizeFn(cand.id)
			if sizeErr != nil {
				if options.SkipMissingBlobs {
					// Mark as nil digest; filtered and logged after wg.Wait.
					metas[idx] = packedBlobMeta{digest: nil, size: 0}
					return
				}

				errOnce.Do(func() {
					firstErr = errors.Wrapf(
						sizeErr,
						"getting size of loose blob %s",
						cand.id,
					)
					cancel()
				})

				return
			}

			metas[idx] = packedBlobMeta{
				digest: cand.digest,
				size:   blobSize,
			}
		}(i, c)
	}

	wg.Wait()

	if firstErr != nil {
		tapNotOk(tw, "collect loose blobs", firstErr)
		return nil, firstErr
	}

	// Filter out skipped blobs (nil digest from SkipMissingBlobs).
	// Emit skip messages here rather than from goroutines to avoid
	// concurrent writes to the TAP writer.
	filtered := metas[:0]
	for i, m := range metas {
		if m.digest != nil {
			filtered = append(filtered, m)
		} else if options.SkipMissingBlobs && i < len(candidates) {
			tapComment(tw, fmt.Sprintf("blob skipped: %s", candidates[i].id))
		}
	}

	metas = filtered

	if len(metas) == 0 {
		return nil, nil
	}

	tapOk(tw, fmt.Sprintf("collect %d loose blobs", len(metas)))

	sort.Slice(metas, func(i, j int) bool {
		return bytes.Compare(metas[i].digest, metas[j].digest) < 0
	})

	return metas, nil
}

func indexPresenceFromV0(index map[string]archiveEntry) map[string]bool {
	m := make(map[string]bool, len(index))
	for k := range index {
		m[k] = true
	}
	return m
}

func indexPresenceFromV1(index map[string]archiveEntryV1) map[string]bool {
	m := make(map[string]bool, len(index))
	for k := range index {
		m[k] = true
	}
	return m
}

// deltaResult holds the output of a parallel delta computation.
// When deltaData is nil, the blob should be stored as a full entry
// (either because delta computation failed or the delta was larger
// than the original).
type deltaResult struct {
	blobIdx   int
	baseIdx   int
	deltaData []byte
}
