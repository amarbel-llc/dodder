package inventory_archive

import (
	"hash/fnv"
)

// LSHBandingSelector finds similar blobs via Locality-Sensitive Hashing
// over MinHash signatures stored in BlobMetadata.Signature. It divides
// each signature into Bands bands of RowsPerBand rows and hashes each
// band into a bucket. Blobs sharing any bucket are candidates. The best
// candidate (highest estimated Jaccard) becomes the delta base.
type LSHBandingSelector struct {
	Bands       int
	RowsPerBand int
	MinBlobSize uint64
	MaxBlobSize uint64
}

var _ BaseSelector = &LSHBandingSelector{}

func (s *LSHBandingSelector) SelectBases(
	blobs BlobSet,
	assignments DeltaAssignments,
) {
	if s.Bands <= 0 || s.RowsPerBand <= 0 {
		return
	}

	n := blobs.Len()
	if n < 2 {
		return
	}

	expectedSigLen := s.Bands * s.RowsPerBand

	// Filter to eligible blobs (right size, has signature of correct length).
	type eligible struct {
		originalIndex int
		signature     []uint32
		size          uint64
	}

	var pool []eligible

	for i := range n {
		meta := blobs.At(i)

		if meta.Size < s.MinBlobSize || meta.Size > s.MaxBlobSize {
			continue
		}

		if len(meta.Signature) != expectedSigLen {
			continue
		}

		pool = append(pool, eligible{
			originalIndex: i,
			signature:     meta.Signature,
			size:          meta.Size,
		})
	}

	if len(pool) < 2 {
		return
	}

	// Build LSH band tables: band index -> band hash -> list of pool indices.
	type bucketKey struct {
		band int
		hash uint64
	}

	buckets := make(map[bucketKey][]int)

	for poolIdx, e := range pool {
		for b := range s.Bands {
			bandStart := b * s.RowsPerBand
			bandEnd := bandStart + s.RowsPerBand
			bh := hashBand(e.signature[bandStart:bandEnd])

			key := bucketKey{band: b, hash: bh}
			buckets[key] = append(buckets[key], poolIdx)
		}
	}

	// For each blob, find candidates and pick the best base.
	for poolIdx, e := range pool {
		candidates := make(map[int]bool)

		for b := range s.Bands {
			bandStart := b * s.RowsPerBand
			bandEnd := bandStart + s.RowsPerBand
			bh := hashBand(e.signature[bandStart:bandEnd])

			key := bucketKey{band: b, hash: bh}

			for _, otherIdx := range buckets[key] {
				if otherIdx != poolIdx {
					candidates[otherIdx] = true
				}
			}
		}

		if len(candidates) == 0 {
			continue
		}

		bestIdx := -1
		bestSim := -1.0

		for candIdx := range candidates {
			sim := MinHashJaccard(e.signature, pool[candIdx].signature)
			if sim > bestSim {
				bestSim = sim
				bestIdx = candIdx
			}
		}

		if bestIdx >= 0 && bestSim > 0 {
			// Assign smaller blob as delta against larger (or first against second
			// if equal). The base should ideally be the larger blob.
			basePoolIdx := bestIdx
			blobPoolIdx := poolIdx

			if pool[blobPoolIdx].size > pool[basePoolIdx].size ||
				(pool[blobPoolIdx].size == pool[basePoolIdx].size && blobPoolIdx > basePoolIdx) {
				// Don't assign a larger blob as delta of a smaller one.
				// For equal-size blobs, break ties by pool index so only
				// the lower index assigns, preventing mutual-delta cycles.
				continue
			}

			assignments.Assign(
				pool[blobPoolIdx].originalIndex,
				pool[basePoolIdx].originalIndex,
			)
		}
	}
}

func hashBand(rows []uint32) uint64 {
	h := fnv.New64a()

	for _, r := range rows {
		b := [4]byte{
			byte(r),
			byte(r >> 8),
			byte(r >> 16),
			byte(r >> 24),
		}
		h.Write(b[:])
	}

	return h.Sum64()
}
