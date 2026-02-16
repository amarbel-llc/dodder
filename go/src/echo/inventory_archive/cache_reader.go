package inventory_archive

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

type CacheReader struct {
	reader       io.ReaderAt
	totalSize    int64
	hashFormatId string
	hashSize     int
	entryCount   uint64
	entriesStart int64
}

func NewCacheReader(
	r io.ReaderAt,
	totalSize int64,
	hashFormatId string,
) (cr *CacheReader, err error) {
	hashSize, err := hashSizeForFormat(hashFormatId)
	if err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	cr = &CacheReader{
		reader:       r,
		totalSize:    totalSize,
		hashFormatId: hashFormatId,
		hashSize:     hashSize,
	}

	if err = cr.readHeader(); err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	return cr, nil
}

func (cr *CacheReader) readHeader() (err error) {
	// magic: 4 bytes
	magic := make([]byte, 4)

	if _, err = cr.reader.ReadAt(magic, 0); err != nil {
		err = errors.Wrapf(err, "reading magic")
		return err
	}

	if string(magic) != CacheFileMagic {
		err = errors.Errorf(
			"invalid magic: got %q, want %q",
			string(magic),
			CacheFileMagic,
		)
		return err
	}

	// version: 2 bytes uint16 BigEndian
	versionBuf := make([]byte, 2)

	if _, err = cr.reader.ReadAt(versionBuf, 4); err != nil {
		err = errors.Wrapf(err, "reading version")
		return err
	}

	version := binary.BigEndian.Uint16(versionBuf)

	if version != CacheFileVersion {
		err = errors.Errorf(
			"unsupported version: got %d, want %d",
			version,
			CacheFileVersion,
		)
		return err
	}

	// hash_format_id_len: 1 byte
	hashFormatIdLenBuf := make([]byte, 1)

	if _, err = cr.reader.ReadAt(hashFormatIdLenBuf, 6); err != nil {
		err = errors.Wrapf(err, "reading hash format id length")
		return err
	}

	hashFormatIdLen := int(hashFormatIdLenBuf[0])

	// hash_format_id: variable
	hashFormatIdBytes := make([]byte, hashFormatIdLen)

	if _, err = cr.reader.ReadAt(hashFormatIdBytes, 7); err != nil {
		err = errors.Wrapf(err, "reading hash format id")
		return err
	}

	fileHashFormatId := string(hashFormatIdBytes)

	if fileHashFormatId != cr.hashFormatId {
		err = errors.Errorf(
			"hash format id mismatch: file has %q, expected %q",
			fileHashFormatId,
			cr.hashFormatId,
		)
		return err
	}

	// entry_count: 8 bytes uint64 BigEndian
	entryCountOffset := int64(7 + hashFormatIdLen)
	entryCountBuf := make([]byte, 8)

	if _, err = cr.reader.ReadAt(
		entryCountBuf,
		entryCountOffset,
	); err != nil {
		err = errors.Wrapf(err, "reading entry count")
		return err
	}

	cr.entryCount = binary.BigEndian.Uint64(entryCountBuf)

	cr.entriesStart = int64(
		4 + // magic
			2 + // version
			1 + // hash_format_id_len
			hashFormatIdLen + // hash_format_id
			8, // entry_count
	)

	return nil
}

func (cr *CacheReader) HashFormatId() string {
	return cr.hashFormatId
}

func (cr *CacheReader) EntryCount() uint64 {
	return cr.entryCount
}

func (cr *CacheReader) entrySize() int64 {
	// hash + archive_checksum + offset + compressed_size
	return int64(cr.hashSize) + int64(cr.hashSize) + 8 + 8
}

func (cr *CacheReader) readEntryAt(index uint64) (
	entry CacheEntry,
	err error,
) {
	offset := cr.entriesStart + int64(index)*cr.entrySize()
	entryBuf := make([]byte, cr.entrySize())

	if _, err = cr.reader.ReadAt(entryBuf, offset); err != nil {
		err = errors.Wrapf(err, "reading entry %d", index)
		return entry, err
	}

	pos := 0

	entry.Hash = make([]byte, cr.hashSize)
	copy(entry.Hash, entryBuf[pos:pos+cr.hashSize])
	pos += cr.hashSize

	entry.ArchiveChecksum = make([]byte, cr.hashSize)
	copy(entry.ArchiveChecksum, entryBuf[pos:pos+cr.hashSize])
	pos += cr.hashSize

	entry.Offset = binary.BigEndian.Uint64(
		entryBuf[pos : pos+8],
	)
	pos += 8

	entry.CompressedSize = binary.BigEndian.Uint64(
		entryBuf[pos : pos+8],
	)

	return entry, nil
}

func (cr *CacheReader) ReadAllEntries() (entries []CacheEntry, err error) {
	entries = make([]CacheEntry, cr.entryCount)

	for i := range cr.entryCount {
		entries[i], err = cr.readEntryAt(i)
		if err != nil {
			err = errors.Wrapf(err, "reading entry %d", i)
			return nil, err
		}
	}

	return entries, nil
}

func (cr *CacheReader) Validate() (err error) {
	checksumOffset := cr.totalSize - int64(cr.hashSize)

	if checksumOffset < 0 {
		err = errors.Errorf(
			"file too small for checksum: %d bytes",
			cr.totalSize,
		)
		return err
	}

	storedChecksum := make([]byte, cr.hashSize)

	if _, err = cr.reader.ReadAt(
		storedChecksum,
		checksumOffset,
	); err != nil {
		err = errors.Wrapf(err, "reading stored checksum")
		return err
	}

	hasher, err := newHashForFormat(cr.hashFormatId)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	contentBuf := make([]byte, checksumOffset)

	if _, err = cr.reader.ReadAt(contentBuf, 0); err != nil {
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

func ToMap(entries []CacheEntry) map[string]CacheEntry {
	m := make(map[string]CacheEntry, len(entries))

	for _, entry := range entries {
		hexHash := hex.EncodeToString(entry.Hash)
		m[hexHash] = entry
	}

	return m
}
