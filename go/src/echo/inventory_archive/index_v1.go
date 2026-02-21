package inventory_archive

import (
	"bytes"
	"encoding/binary"
	"io"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

func WriteIndexV1(
	w io.Writer,
	hashFormatId string,
	entries []IndexEntryV1,
) (checksum []byte, err error) {
	hasher, err := newHashForFormat(hashFormatId)
	if err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	hashSize, err := hashSizeForFormat(hashFormatId)
	if err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	if err = verifySortedV1(entries, hashSize); err != nil {
		return nil, err
	}

	multiWriter := io.MultiWriter(w, hasher)

	if err = writeIndexV1Header(
		multiWriter,
		hashFormatId,
		uint64(len(entries)),
	); err != nil {
		return nil, err
	}

	if err = writeIndexV1FanOut(multiWriter, entries); err != nil {
		return nil, err
	}

	if err = writeIndexV1Entries(multiWriter, entries); err != nil {
		return nil, err
	}

	checksum = hasher.Sum(nil)

	if _, err = w.Write(checksum); err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	return checksum, nil
}

func verifySortedV1(entries []IndexEntryV1, hashSize int) error {
	for i := 1; i < len(entries); i++ {
		if bytes.Compare(entries[i-1].Hash, entries[i].Hash) >= 0 {
			return errors.Errorf(
				"entries not sorted: entry %d >= entry %d",
				i-1,
				i,
			)
		}
	}

	for i, entry := range entries {
		if len(entry.Hash) != hashSize {
			return errors.Errorf(
				"entry %d: hash length %d != expected %d",
				i,
				len(entry.Hash),
				hashSize,
			)
		}
	}

	return nil
}

func writeIndexV1Header(
	w io.Writer,
	hashFormatId string,
	entryCount uint64,
) (err error) {
	// magic: 4 bytes
	if _, err = w.Write([]byte(IndexFileMagic)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// version: 2 bytes uint16 BigEndian
	if err = binary.Write(
		w,
		binary.BigEndian,
		IndexFileVersionV1,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// hash_format_id_len: 1 byte
	hashFormatIdBytes := []byte(hashFormatId)

	if len(hashFormatIdBytes) > 255 {
		err = errors.Errorf(
			"hash format id too long: %d bytes",
			len(hashFormatIdBytes),
		)
		return err
	}

	if _, err = w.Write(
		[]byte{byte(len(hashFormatIdBytes))},
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// hash_format_id: variable
	if _, err = w.Write(hashFormatIdBytes); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// entry_count: 8 bytes uint64 BigEndian
	if err = binary.Write(
		w,
		binary.BigEndian,
		entryCount,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return nil
}

func writeIndexV1FanOut(
	w io.Writer,
	entries []IndexEntryV1,
) (err error) {
	var fanOut [256]uint64

	for _, entry := range entries {
		firstByte := entry.Hash[0]
		for j := int(firstByte); j < 256; j++ {
			fanOut[j]++
		}
	}

	for i := range 256 {
		if err = binary.Write(
			w,
			binary.BigEndian,
			fanOut[i],
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return nil
}

func writeIndexV1Entries(
	w io.Writer,
	entries []IndexEntryV1,
) (err error) {
	for i, entry := range entries {
		// hash: N bytes
		if _, err = w.Write(entry.Hash); err != nil {
			err = errors.Wrapf(err, "writing entry %d hash", i)
			return err
		}

		// pack_offset: 8 bytes uint64 BigEndian
		if err = binary.Write(
			w,
			binary.BigEndian,
			entry.PackOffset,
		); err != nil {
			err = errors.Wrapf(err, "writing entry %d pack offset", i)
			return err
		}

		// compressed_size: 8 bytes uint64 BigEndian
		if err = binary.Write(
			w,
			binary.BigEndian,
			entry.CompressedSize,
		); err != nil {
			err = errors.Wrapf(err, "writing entry %d compressed size", i)
			return err
		}

		// entry_type: 1 byte
		if _, err = w.Write([]byte{entry.EntryType}); err != nil {
			err = errors.Wrapf(err, "writing entry %d entry type", i)
			return err
		}

		// base_offset: 8 bytes uint64 BigEndian
		if err = binary.Write(
			w,
			binary.BigEndian,
			entry.BaseOffset,
		); err != nil {
			err = errors.Wrapf(err, "writing entry %d base offset", i)
			return err
		}
	}

	return nil
}

type IndexReaderV1 struct {
	reader       io.ReaderAt
	totalSize    int64
	hashFormatId string
	hashSize     int
	entryCount   uint64
	fanOut       [256]uint64
	entriesStart int64
}

func NewIndexReaderV1(
	r io.ReaderAt,
	totalSize int64,
	hashFormatId string,
) (ir *IndexReaderV1, err error) {
	hashSize, err := hashSizeForFormat(hashFormatId)
	if err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	ir = &IndexReaderV1{
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

func (ir *IndexReaderV1) readHeader() (err error) {
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

	if version != IndexFileVersionV1 {
		err = errors.Errorf(
			"unsupported version: got %d, want %d",
			version,
			IndexFileVersionV1,
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

func (ir *IndexReaderV1) readFanOut() (err error) {
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

func (ir *IndexReaderV1) headerSize() int64 {
	return int64(
		4 + // magic
			2 + // version
			1 + // hash_format_id_len
			len(ir.hashFormatId) + // hash_format_id
			8, // entry_count
	)
}

func (ir *IndexReaderV1) entrySize() int64 {
	return int64(ir.hashSize) + 8 + 8 + 1 + 8 // hash + pack_offset + compressed_size + entry_type + base_offset
}

func (ir *IndexReaderV1) HashFormatId() string {
	return ir.hashFormatId
}

func (ir *IndexReaderV1) EntryCount() uint64 {
	return ir.entryCount
}

func (ir *IndexReaderV1) FanOut() [256]uint64 {
	return ir.fanOut
}

func (ir *IndexReaderV1) readEntryAt(index uint64) (
	entry IndexEntryV1,
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

	pos := ir.hashSize

	entry.PackOffset = binary.BigEndian.Uint64(
		entryBuf[pos : pos+8],
	)
	pos += 8

	entry.CompressedSize = binary.BigEndian.Uint64(
		entryBuf[pos : pos+8],
	)
	pos += 8

	entry.EntryType = entryBuf[pos]
	pos++

	entry.BaseOffset = binary.BigEndian.Uint64(
		entryBuf[pos : pos+8],
	)

	return entry, nil
}

func (ir *IndexReaderV1) LookupHash(hash []byte) (
	packOffset uint64,
	compressedSize uint64,
	entryType byte,
	baseOffset uint64,
	found bool,
	err error,
) {
	if ir.entryCount == 0 {
		return 0, 0, 0, 0, false, nil
	}

	firstByte := hash[0]

	// Determine search range from fan-out table
	var lo uint64

	if firstByte > 0 {
		lo = ir.fanOut[firstByte-1]
	}

	hi := ir.fanOut[firstByte]

	if lo >= hi {
		return 0, 0, 0, 0, false, nil
	}

	// Binary search within the range
	for lo < hi {
		mid := lo + (hi-lo)/2

		entry, readErr := ir.readEntryAt(mid)
		if readErr != nil {
			return 0, 0, 0, 0, false, readErr
		}

		cmp := bytes.Compare(hash, entry.Hash)

		switch {
		case cmp == 0:
			return entry.PackOffset, entry.CompressedSize, entry.EntryType, entry.BaseOffset, true, nil
		case cmp < 0:
			hi = mid
		default:
			lo = mid + 1
		}
	}

	return 0, 0, 0, 0, false, nil
}

func (ir *IndexReaderV1) ReadAllEntries() (entries []IndexEntryV1, err error) {
	entries = make([]IndexEntryV1, ir.entryCount)

	for i := range ir.entryCount {
		entries[i], err = ir.readEntryAt(i)
		if err != nil {
			err = errors.Wrapf(err, "reading entry %d", i)
			return nil, err
		}
	}

	return entries, nil
}

func (ir *IndexReaderV1) Validate() (err error) {
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
