# Blob Similarity Algorithms Research

Research into algorithms for sorting and grouping binary blobs by content
similarity, specifically for delta compression base selection in inventory
archives.

## Context

The inventory archive packer needs to choose delta bases: given N blobs, which
pairs will produce the smallest deltas? The current `SizeBasedSelector` uses
blob size as a proxy for similarity. This document surveys algorithms that use
actual content signals.

## Git's Approach: Heuristic Sort + Sliding Window

Git sorts objects before delta compression using a multi-key sort:

1. **Object type** (commit, tree, blob, tag) — deltas only within same type
2. **Base name** (filename component) — assumes path-to-content affinity
3. **File size descending** — larger objects first (deltas reference larger bases)
4. **Most recent modification time first** — newest version as base for read
   performance

After sorting, a sliding window of size W (default 10, configurable via
`pack.window`) tries each object against its W nearest neighbors. The best delta
(smallest result) wins.

Git's custom delta algorithm indexes the base object with 16-byte-aligned chunk
fingerprints, then scans the target looking for matches. Output is COPY
(offset+length from base) and INSERT (literal bytes) instructions.

**Applicability to dodder**: Limited. Git relies heavily on path names for
grouping. Content-addressed blob stores lack stable path associations. Size-only
sorting is too coarse. The sliding window concept is reusable but needs
content-based ordering.

## Content-Defined Chunking (CDC) + Rolling Hashes

### Rabin Fingerprints

Polynomial hash over a sliding window of w bytes in GF(2):

```
fp(b_1,...,b_w) = (b_1 * p^(w-1) + ... + b_w) mod q
```

Sliding update is O(1): `fp_new = (fp_old * p - b_1 * p^w + b_{w+1}) mod q`.

### Gear Hash (FastCDC)

Simpler and faster rolling hash:

```
fp = (fp << 1) + GEAR_TABLE[byte]
```

`GEAR_TABLE` is 256 precomputed random 64-bit values. Chunk boundary when
`fp & MASK == 0`. FastCDC adds normalized chunking (two-level mask) and minimum
chunk skip.

Approximately 3-10x faster than Rabin fingerprinting. ~1-2 GB/s on modern
hardware.

### CDC for Similarity Estimation

Split each blob into variable-length chunks at content-defined boundaries.
Insertions/deletions only affect nearby chunks. Two versions of a note sharing
90% content will share ~90% of chunk hashes.

Similarity via Jaccard over chunk hash sets:

```
similarity(A, B) = |chunks(A) intersection chunks(B)| / |chunks(A) union chunks(B)|
```

**Memory**: For a 1KB blob with 64-byte avg chunks: ~16 chunk hashes at 8
bytes each = 128 bytes/blob. For 1M blobs: ~128 MB.

**Used by**: Borg Backup (Buzhash), Restic (Rabin-based) — primarily for exact
chunk deduplication rather than similarity estimation.

## MinHash

Estimates Jaccard similarity between sets. The probability that
`min(h(A)) == min(h(B))` equals `J(A,B)`.

### Algorithm

1. Represent each blob as a set of features (CDC chunk hashes or byte n-grams)
2. Choose k independent hash functions h_1,...,h_k
3. For each blob, signature = `[min(h_i(x) for x in features) for i in 1..k]`
4. Estimated similarity = fraction of positions where signatures agree

### One-Permutation Hashing

Instead of k independent hash functions, use one hash and partition the hash
space into k bins. Keep the minimum per bin. Faster, similar accuracy.

### Feature Set Choices

- **Byte n-grams** (e.g., 4-grams): Fine-grained. A 1KB blob has ~1021
  4-grams. Captures local byte patterns. A 5% edit changes ~20% of 4-grams.
- **CDC chunk hashes**: Coarser-grained. Better for structural edits (insert a
  paragraph). A 5% edit changes ~5% of chunks. Best for zettelkasten edits.
- **Line shingles**: Assumes line structure. Not suitable for binary blobs.

### Memory and Accuracy

With k=128: 512 bytes/blob, standard error ~0.035 at similarity 0.8.
With k=64: 256 bytes/blob, standard error ~0.050 at similarity 0.8.
Sufficient for grouping — exact similarity not needed.

## SimHash (Charikar)

Produces a single fixed-width hash (typically 64 bits) where Hamming distance
approximates content dissimilarity.

### Algorithm

1. Extract weighted features from blob
2. Initialize b-dimensional vector V to 0
3. For each feature f with weight w: hash f to b bits, add w to V[i] where
   bit i is 1, subtract w where 0
4. SimHash = bit vector where V[i] > 0

### Near-Neighbor Search

Finding Hamming-distance neighbors requires multiple tables with permuted bit
positions. For d-bit hashes with up to k differences: C(d, k/2) tables. Gets
expensive for large k.

**Memory**: 8 bytes/blob. Extremely compact (8 MB for 1M blobs).

**Tradeoff vs MinHash**: Far less memory but coarser signal. Good for
near-duplicate detection (Hamming distance <= 3 out of 64), weak for moderate
similarity (50-80% overlap).

## Feature Extraction Approaches

### Byte Frequency Histogram

Count frequency of each byte value (0-255). Compare via cosine similarity.

- Memory: 256-1024 bytes/blob
- Throughput: ~5-10 GB/s (memory-bound)
- Weakness: All English text has similar byte distributions. Two completely
  different notes may score as similar. Useful only as a pre-filter.

### Hashed N-gram Vectors

Count byte n-grams, hash into D buckets (feature hashing trick). Compare via
cosine similarity.

- With D=256 and 4-grams: 512 bytes/blob
- Better than byte frequency — captures local patterns
- Still loses ordering information

### Sample-Based Fingerprinting

Extract short sequences at deterministic positions in the blob.

- Memory: s * 8 bytes/blob (s=16 gives 128 bytes)
- Fragile to insertions — positions shift. Only detects near-exact duplicates.

## Comparison Table

| Approach              | Per-blob  | 1M blobs | Throughput   | Small edits | Binary |
|-----------------------|-----------|----------|--------------|-------------|--------|
| Git sort (path+size)  | ~64 B     | ~64 MB   | O(n log n)   | N/A         | N/A    |
| Byte histogram        | 256-1024 B| 256 MB-1G| ~5-10 GB/s   | Poor        | Moderate|
| N-gram vector (D=256) | 512 B     | 512 MB   | ~2-5 GB/s    | Moderate    | Moderate|
| CDC chunk hashes      | ~128 B    | 128 MB   | ~1-2 GB/s    | Excellent   | Good   |
| MinHash (k=128)       | 512 B     | 512 MB   | ~100-500 MB/s| Excellent   | Good   |
| MinHash (k=64)        | 256 B     | 256 MB   | ~500 MB/s-1G | Excellent   | Good   |
| SimHash (64-bit)      | 8 B       | 8 MB     | ~1-3 GB/s    | Good        | Moderate|
| Sample fingerprint    | 128 B     | 128 MB   | Very fast    | Poor        | Moderate|

## LSH Banding

General technique for sublinear candidate finding with MinHash:

1. Compute MinHash signature of k values per blob
2. Divide into b bands of r rows (k = b * r)
3. Per band, hash r values into a bucket
4. Blobs sharing any bucket are candidate pairs

Probability of candidacy at Jaccard similarity s: `P = 1 - (1 - s^r)^b`

Tuning b and r controls the similarity threshold:

| b  | r | k   | threshold |
|----|---|-----|-----------|
| 16 | 4 | 64  | ~0.50     |
| 20 | 5 | 100 | ~0.55     |
| 32 | 4 | 128 | ~0.42     |
| 10 | 5 | 50  | ~0.63     |

## Recommendation

For dodder's zettelkasten workload (small edits to text blobs, content-addressed
without path metadata, tens of thousands growing to hundreds of thousands):

**MinHash over CDC chunk hashes with LSH banding.**

- CDC (Gear hash) handles small edits excellently — content-defined boundaries
  mean only chunks near the edit change
- MinHash provides accurate Jaccard estimation at low memory cost
- LSH banding gives sublinear candidate finding, scaling to hundreds of
  thousands
- The feature extraction (CDC vs n-grams) and the similarity search (LSH
  banding vs pairwise) are independently swappable via the `SignatureComputer`
  and `BaseSelector` interface split

See `2026-02-23-delta-similarity-design.md` for the implementation design.
