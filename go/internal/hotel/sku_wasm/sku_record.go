package sku_wasm

import (
	"context"
	"encoding/binary"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/wasm"
)

// SkuRecordSize is the byte size of the flat SKU record layout:
// 5 strings (ptr+len each = 2*4 = 8 bytes) + 2 lists (ptr+count each = 8
// bytes) = 7 * 8 = 56 bytes.
const SkuRecordSize = 7 * 8

// WriteSkuRecord writes the flat record layout for a canonical ABI sku record.
// Fields are laid out sequentially: genre, object-id, type, tags,
// tags-implicit, blob-digest, description. Each string is (ptr: u32, len: u32)
// and each list is (ptr: u32, count: u32).
func WriteSkuRecord(
	alloc *wasm.BumpAllocator,
	genre, objectId, tipe string,
	tags, tagsImplicit []string,
	blobDigest, description string,
) uint32 {
	// Write all data first
	genrePtr, genreLen := wasm.WriteString(alloc, genre)
	objectIdPtr, objectIdLen := wasm.WriteString(alloc, objectId)
	tipePtr, tipeLen := wasm.WriteString(alloc, tipe)
	tagsPtr, tagsCount := wasm.WriteStringList(alloc, tags)
	tagsImplicitPtr, tagsImplicitCount := wasm.WriteStringList(alloc, tagsImplicit)
	blobDigestPtr, blobDigestLen := wasm.WriteString(alloc, blobDigest)
	descriptionPtr, descriptionLen := wasm.WriteString(alloc, description)

	// Write the flat record struct
	recordPtr := alloc.Alloc(SkuRecordSize, 4)
	m := alloc.Memory()[recordPtr:]

	fields := []uint32{
		genrePtr, genreLen,
		objectIdPtr, objectIdLen,
		tipePtr, tipeLen,
		tagsPtr, tagsCount,
		tagsImplicitPtr, tagsImplicitCount,
		blobDigestPtr, blobDigestLen,
		descriptionPtr, descriptionLen,
	}

	for i, v := range fields {
		binary.LittleEndian.PutUint32(m[i*4:], v)
	}

	return recordPtr
}

// adjustSkuPointers adds basePtr to all pointer fields in a serialized SKU
// record and its list element arrays so that offsets relative to 0 become
// absolute WASM memory addresses.
func adjustSkuPointers(buf []byte, recordOffset, basePtr uint32) {
	// Record has 14 u32 fields (7 pairs). The pointer fields are at even
	// indices: 0, 2, 4, 6, 8, 10, 12.
	ptrFieldIndices := [7]int{0, 2, 4, 6, 8, 10, 12}
	for _, idx := range ptrFieldIndices {
		off := recordOffset + uint32(idx)*4
		v := binary.LittleEndian.Uint32(buf[off:])
		binary.LittleEndian.PutUint32(buf[off:], v+basePtr)
	}

	// For the two list fields (tags at record field pair 3, tags-implicit at
	// pair 4), each list element is a (ptr, len) pair where the ptr also
	// needs adjustment.
	for _, pairIdx := range []int{3, 4} {
		// After the record pointer adjustment above, the list ptr is now
		// absolute. Read count from the record (odd field = count).
		listPtrOff := recordOffset + uint32(pairIdx*2)*4
		countOff := recordOffset + uint32(pairIdx*2+1)*4

		// The list ptr was just adjusted above, so subtract basePtr to get
		// the buffer-local offset for reading element data.
		listPtrAbsolute := binary.LittleEndian.Uint32(buf[listPtrOff:])
		listPtrLocal := listPtrAbsolute - basePtr
		count := binary.LittleEndian.Uint32(buf[countOff:])

		for i := uint32(0); i < count; i++ {
			elemOff := listPtrLocal + i*8
			v := binary.LittleEndian.Uint32(buf[elemOff:])
			binary.LittleEndian.PutUint32(buf[elemOff:], v+basePtr)
		}
	}
}

// MarshalSkuToModule writes an SKU record into a WASM module's linear memory
// using the guest-exported cabi_realloc for allocation. Returns the pointer to
// the record in WASM memory.
func MarshalSkuToModule(
	ctx context.Context,
	mod *wasm.Module,
	genre, objectId, tipe string,
	tags, tagsImplicit []string,
	blobDigest, description string,
) (recordPtr uint32, err error) {
	totalStringBytes := uint32(len(genre) + len(objectId) + len(tipe) +
		len(blobDigest) + len(description))

	for _, t := range tags {
		totalStringBytes += uint32(len(t))
	}

	for _, t := range tagsImplicit {
		totalStringBytes += uint32(len(t))
	}

	// Each string in a list needs an 8-byte (ptr, len) element entry.
	listOverhead := uint32((len(tags) + len(tagsImplicit)) * 8)

	// Total: string data + list element arrays + record struct + alignment
	// padding headroom.
	totalSize := totalStringBytes + listOverhead + SkuRecordSize + 64

	// Write to a temporary buffer at offset 0.
	buf := make([]byte, totalSize)
	alloc := wasm.MakeBumpAllocator(buf, 0)

	recordOffset := WriteSkuRecord(
		&alloc, genre, objectId, tipe,
		tags, tagsImplicit, blobDigest, description,
	)

	usedBytes := alloc.Offset()

	// Allocate a block in WASM memory via the guest's cabi_realloc.
	basePtr, allocErr := mod.CallCabiRealloc(ctx, 0, 0, 4, usedBytes)
	if allocErr != nil {
		return 0, errors.Wrap(allocErr)
	}

	// Adjust all pointer fields so they become absolute WASM addresses.
	adjustSkuPointers(buf, recordOffset, basePtr)

	// Copy the used portion of the buffer into WASM memory.
	if !mod.WriteBytes(basePtr, buf[:usedBytes]) {
		return 0, errors.ErrorWithStackf(
			"failed to write %d bytes to WASM memory at offset %d",
			usedBytes, basePtr,
		)
	}

	return recordOffset + basePtr, nil
}
