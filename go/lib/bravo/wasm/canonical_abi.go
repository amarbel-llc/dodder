package wasm

import (
	"context"
	"encoding/binary"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
)

// BumpAllocator is a simple arena allocator over a byte slice. Mirrors the
// canonical ABI's cabi_realloc pattern but operates on the host side for
// writing data before copying to WASM memory.
type BumpAllocator struct {
	memory []byte
	offset uint32
}

func MakeBumpAllocator(memory []byte, offset uint32) BumpAllocator {
	return BumpAllocator{memory: memory, offset: offset}
}

func (a *BumpAllocator) Alloc(size, align uint32) uint32 {
	// Align up
	a.offset = (a.offset + align - 1) &^ (align - 1)
	ptr := a.offset
	a.offset += size
	return ptr
}

// WriteString writes a UTF-8 string into the allocator's memory and returns
// the (ptr, len) pair for canonical ABI string representation.
func WriteString(alloc *BumpAllocator, s string) (ptr, length uint32) {
	length = uint32(len(s))
	ptr = alloc.Alloc(length, 1)
	copy(alloc.memory[ptr:ptr+length], s)
	return ptr, length
}

// ReadString reads a canonical ABI string from memory at (ptr, length).
func ReadString(memory []byte, ptr, length uint32) string {
	return string(memory[ptr : ptr+length])
}

// WriteStringList writes a list<string> in canonical ABI format. Each element
// is a (ptr: u32, len: u32) pair. Returns the pointer to the list header and
// the element count.
func WriteStringList(
	alloc *BumpAllocator,
	strings []string,
) (listPtr, count uint32) {
	count = uint32(len(strings))

	// First, write all string data and collect (ptr, len) pairs
	type stringPair struct{ ptr, length uint32 }
	pairs := make([]stringPair, count)
	for i, s := range strings {
		pairs[i].ptr, pairs[i].length = WriteString(alloc, s)
	}

	// Then write the list elements: array of (ptr: u32, len: u32)
	listPtr = alloc.Alloc(count*8, 4)
	for i, p := range pairs {
		offset := listPtr + uint32(i)*8
		binary.LittleEndian.PutUint32(alloc.memory[offset:], p.ptr)
		binary.LittleEndian.PutUint32(alloc.memory[offset+4:], p.length)
	}

	return listPtr, count
}

// ReadStringList reads a canonical ABI list<string> from memory.
func ReadStringList(
	memory []byte,
	listPtr, count uint32,
) []string {
	result := make([]string, count)
	for i := uint32(0); i < count; i++ {
		offset := listPtr + i*8
		ptr := binary.LittleEndian.Uint32(memory[offset:])
		length := binary.LittleEndian.Uint32(memory[offset+4:])
		result[i] = ReadString(memory, ptr, length)
	}
	return result
}

// SkuRecordSize is the byte size of the flat SKU record layout:
// 5 strings (ptr+len each = 2*4 = 8 bytes) + 2 lists (ptr+count each = 8
// bytes) = 7 * 8 = 56 bytes.
const SkuRecordSize = 7 * 8

// WriteSkuRecord writes the flat record layout for a canonical ABI sku record.
// Fields are laid out sequentially: genre, object-id, type, tags,
// tags-implicit, blob-digest, description. Each string is (ptr: u32, len: u32)
// and each list is (ptr: u32, count: u32).
func WriteSkuRecord(
	alloc *BumpAllocator,
	genre, objectId, tipe string,
	tags, tagsImplicit []string,
	blobDigest, description string,
) uint32 {
	// Write all data first
	genrePtr, genreLen := WriteString(alloc, genre)
	objectIdPtr, objectIdLen := WriteString(alloc, objectId)
	tipePtr, tipeLen := WriteString(alloc, tipe)
	tagsPtr, tagsCount := WriteStringList(alloc, tags)
	tagsImplicitPtr, tagsImplicitCount := WriteStringList(alloc, tagsImplicit)
	blobDigestPtr, blobDigestLen := WriteString(alloc, blobDigest)
	descriptionPtr, descriptionLen := WriteString(alloc, description)

	// Write the flat record struct
	recordPtr := alloc.Alloc(SkuRecordSize, 4)
	m := alloc.memory[recordPtr:]

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
	mod *Module,
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
	alloc := MakeBumpAllocator(buf, 0)

	recordOffset := WriteSkuRecord(
		&alloc, genre, objectId, tipe,
		tags, tagsImplicit, blobDigest, description,
	)

	usedBytes := alloc.offset

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
