package inventory_archive

import (
	"bytes"
	"encoding/binary"
	"io"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

type IndexReader struct {
	reader       io.ReaderAt
	totalSize    int64
	hashFormatId string
	hashSize     int
	entryCount   uint64
	fanOut       [256]uint64
	entriesStart int64
}

func NewIndexReader(
	r io.ReaderAt,
	totalSize int64,
	hashFormatId string,
) (ir *IndexReader, err error) {
	hashSize, err := hashSizeForFormat(hashFormatId)
	if err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	ir = &IndexReader{
		reader:       r,
		totalSize:    totalSize,
		hashFormatId: hashFormatId,
		hashSize:     hashSize,
	}

	if err = ir.readHeader(); err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	if err = ir.readFanOut(); err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	return ir, nil
}

func (ir *IndexReader) readHeader() (err error) {
	// magic: 4 bytes
	magic := make([]byte, 4)

	if _, err = ir.reader.ReadAt(magic, 0); err != nil {
		err = errors.Wrapf(err, "reading magic")
		return err
	}

	if string(magic) != IndexFileMagic {
		err = errors.Errorf(
			"invalid magic: got %q, want %q",
			string(magic),
			IndexFileMagic,
		)
		return err
	}

	// version: 2 bytes uint16 BigEndian
	versionBuf := make([]byte, 2)

	if _, err = ir.reader.ReadAt(versionBuf, 4); err != nil {
		err = errors.Wrapf(err, "reading version")
		return err
	}

	version := binary.BigEndian.Uint16(versionBuf)

	if version != IndexFileVersion {
		err = errors.Errorf(
			"unsupported version: got %d, want %d",
			version,
			IndexFileVersion,
		)
		return err
	}

	// hash_format_id_len: 1 byte
	hashFormatIdLenBuf := make([]byte, 1)

	if _, err = ir.reader.ReadAt(hashFormatIdLenBuf, 6); err != nil {
		err = errors.Wrapf(err, "reading hash format id length")
		return err
	}

	hashFormatIdLen := int(hashFormatIdLenBuf[0])

	// hash_format_id: variable
	hashFormatIdBytes := make([]byte, hashFormatIdLen)

	if _, err = ir.reader.ReadAt(hashFormatIdBytes, 7); err != nil {
		err = errors.Wrapf(err, "reading hash format id")
		return err
	}

	fileHashFormatId := string(hashFormatIdBytes)

	if fileHashFormatId != ir.hashFormatId {
		err = errors.Errorf(
			"hash format id mismatch: file has %q, expected %q",
			fileHashFormatId,
			ir.hashFormatId,
		)
		return err
	}

	// entry_count: 8 bytes uint64 BigEndian
	entryCountOffset := int64(7 + hashFormatIdLen)
	entryCountBuf := make([]byte, 8)

	if _, err = ir.reader.ReadAt(
		entryCountBuf,
		entryCountOffset,
	); err != nil {
		err = errors.Wrapf(err, "reading entry count")
		return err
	}

	ir.entryCount = binary.BigEndian.Uint64(entryCountBuf)

	return nil
}

func (ir *IndexReader) readFanOut() (err error) {
	headerSize := ir.headerSize()
	fanOutBuf := make([]byte, 256*8)

	if _, err = ir.reader.ReadAt(fanOutBuf, headerSize); err != nil {
		err = errors.Wrapf(err, "reading fan-out table")
		return err
	}

	for i := range 256 {
		ir.fanOut[i] = binary.BigEndian.Uint64(
			fanOutBuf[i*8 : (i+1)*8],
		)
	}

	ir.entriesStart = headerSize + 256*8

	return nil
}

func (ir *IndexReader) headerSize() int64 {
	return int64(
		4 + // magic
			2 + // version
			1 + // hash_format_id_len
			len(ir.hashFormatId) + // hash_format_id
			8, // entry_count
	)
}

func (ir *IndexReader) entrySize() int64 {
	return int64(ir.hashSize) + 8 + 8 // hash + pack_offset + compressed_size
}

func (ir *IndexReader) HashFormatId() string {
	return ir.hashFormatId
}

func (ir *IndexReader) EntryCount() uint64 {
	return ir.entryCount
}

func (ir *IndexReader) FanOut() [256]uint64 {
	return ir.fanOut
}

func (ir *IndexReader) readEntryAt(index uint64) (
	entry IndexEntry,
	err error,
) {
	offset := ir.entriesStart + int64(index)*ir.entrySize()
	entryBuf := make([]byte, ir.entrySize())

	if _, err = ir.reader.ReadAt(entryBuf, offset); err != nil {
		err = errors.Wrapf(err, "reading entry %d", index)
		return entry, err
	}

	entry.Hash = make([]byte, ir.hashSize)
	copy(entry.Hash, entryBuf[:ir.hashSize])

	entry.PackOffset = binary.BigEndian.Uint64(
		entryBuf[ir.hashSize : ir.hashSize+8],
	)

	entry.CompressedSize = binary.BigEndian.Uint64(
		entryBuf[ir.hashSize+8 : ir.hashSize+16],
	)

	return entry, nil
}

func (ir *IndexReader) LookupHash(hash []byte) (
	packOffset uint64,
	compressedSize uint64,
	found bool,
	err error,
) {
	if ir.entryCount == 0 {
		return 0, 0, false, nil
	}

	firstByte := hash[0]

	// Determine search range from fan-out table
	var lo uint64

	if firstByte > 0 {
		lo = ir.fanOut[firstByte-1]
	}

	hi := ir.fanOut[firstByte]

	if lo >= hi {
		return 0, 0, false, nil
	}

	// Binary search within the range
	for lo < hi {
		mid := lo + (hi-lo)/2

		entry, readErr := ir.readEntryAt(mid)
		if readErr != nil {
			return 0, 0, false, readErr
		}

		cmp := bytes.Compare(hash, entry.Hash)

		switch {
		case cmp == 0:
			return entry.PackOffset, entry.CompressedSize, true, nil
		case cmp < 0:
			hi = mid
		default:
			lo = mid + 1
		}
	}

	return 0, 0, false, nil
}

func (ir *IndexReader) ReadAllEntries() (entries []IndexEntry, err error) {
	entries = make([]IndexEntry, ir.entryCount)

	for i := range ir.entryCount {
		entries[i], err = ir.readEntryAt(i)
		if err != nil {
			err = errors.Wrapf(err, "reading entry %d", i)
			return nil, err
		}
	}

	return entries, nil
}

func (ir *IndexReader) Validate() (err error) {
	// The checksum is the last hashSize bytes of the file.
	// It covers everything before it.
	checksumOffset := ir.totalSize - int64(ir.hashSize)

	if checksumOffset < 0 {
		err = errors.Errorf(
			"file too small for checksum: %d bytes",
			ir.totalSize,
		)
		return err
	}

	storedChecksum := make([]byte, ir.hashSize)

	if _, err = ir.reader.ReadAt(
		storedChecksum,
		checksumOffset,
	); err != nil {
		err = errors.Wrapf(err, "reading stored checksum")
		return err
	}

	hasher, err := newHashForFormat(ir.hashFormatId)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	// Read everything before the checksum and hash it
	contentBuf := make([]byte, checksumOffset)

	if _, err = ir.reader.ReadAt(contentBuf, 0); err != nil {
		err = errors.Wrapf(err, "reading content for hashing")
		return err
	}

	if _, err = hasher.Write(contentBuf); err != nil {
		err = errors.Wrapf(err, "hashing content")
		return err
	}

	computedChecksum := hasher.Sum(nil)

	if !bytes.Equal(computedChecksum, storedChecksum) {
		err = errors.Errorf(
			"checksum mismatch: stored %x, computed %x",
			storedChecksum,
			computedChecksum,
		)
		return err
	}

	return nil
}
