package inventory_archive

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

func WriteCacheV1(
	w io.Writer,
	hashFormatId string,
	entries []CacheEntryV1,
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

	if err = verifyCacheV1Sorted(entries, hashSize); err != nil {
		return nil, err
	}

	multiWriter := io.MultiWriter(w, hasher)

	if err = writeCacheV1Header(
		multiWriter,
		hashFormatId,
		uint64(len(entries)),
	); err != nil {
		return nil, err
	}

	if err = writeCacheV1Entries(multiWriter, entries); err != nil {
		return nil, err
	}

	checksum = hasher.Sum(nil)

	if _, err = w.Write(checksum); err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	return checksum, nil
}

func verifyCacheV1Sorted(entries []CacheEntryV1, hashSize int) error {
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

		if len(entry.ArchiveChecksum) != hashSize {
			return errors.Errorf(
				"entry %d: archive checksum length %d != expected %d",
				i,
				len(entry.ArchiveChecksum),
				hashSize,
			)
		}
	}

	return nil
}

func writeCacheV1Header(
	w io.Writer,
	hashFormatId string,
	entryCount uint64,
) (err error) {
	// magic: 4 bytes
	if _, err = w.Write([]byte(CacheFileMagic)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// version: 2 bytes uint16 BigEndian
	if err = binary.Write(
		w,
		binary.BigEndian,
		CacheFileVersionV1,
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

func writeCacheV1Entries(
	w io.Writer,
	entries []CacheEntryV1,
) (err error) {
	for i, entry := range entries {
		// hash: N bytes
		if _, err = w.Write(entry.Hash); err != nil {
			err = errors.Wrapf(err, "writing entry %d hash", i)
			return err
		}

		// archive_checksum: N bytes
		if _, err = w.Write(entry.ArchiveChecksum); err != nil {
			err = errors.Wrapf(err, "writing entry %d archive checksum", i)
			return err
		}

		// offset: 8 bytes uint64 BigEndian
		if err = binary.Write(
			w,
			binary.BigEndian,
			entry.Offset,
		); err != nil {
			err = errors.Wrapf(err, "writing entry %d offset", i)
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

type CacheReaderV1 struct {
	reader       io.ReaderAt
	totalSize    int64
	hashFormatId string
	hashSize     int
	entryCount   uint64
	entriesStart int64
}

func NewCacheReaderV1(
	r io.ReaderAt,
	totalSize int64,
	hashFormatId string,
) (cr *CacheReaderV1, err error) {
	hashSize, err := hashSizeForFormat(hashFormatId)
	if err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	cr = &CacheReaderV1{
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

func (cr *CacheReaderV1) readHeader() (err error) {
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

	if version != CacheFileVersionV1 {
		err = errors.Errorf(
			"unsupported version: got %d, want %d",
			version,
			CacheFileVersionV1,
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

func (cr *CacheReaderV1) HashFormatId() string {
	return cr.hashFormatId
}

func (cr *CacheReaderV1) EntryCount() uint64 {
	return cr.entryCount
}

func (cr *CacheReaderV1) entrySize() int64 {
	// hash + archive_checksum + offset + compressed_size + entry_type + base_offset
	return int64(cr.hashSize) + int64(cr.hashSize) + 8 + 8 + 1 + 8
}

func (cr *CacheReaderV1) readEntryAt(index uint64) (
	entry CacheEntryV1,
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
	pos += 8

	entry.EntryType = entryBuf[pos]
	pos += 1

	entry.BaseOffset = binary.BigEndian.Uint64(
		entryBuf[pos : pos+8],
	)

	return entry, nil
}

func (cr *CacheReaderV1) ReadAllEntries() (entries []CacheEntryV1, err error) {
	entries = make([]CacheEntryV1, cr.entryCount)

	for i := range cr.entryCount {
		entries[i], err = cr.readEntryAt(i)
		if err != nil {
			err = errors.Wrapf(err, "reading entry %d", i)
			return nil, err
		}
	}

	return entries, nil
}

func (cr *CacheReaderV1) Validate() (err error) {
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

func ToMapV1(entries []CacheEntryV1) map[string]CacheEntryV1 {
	m := make(map[string]CacheEntryV1, len(entries))

	for _, entry := range entries {
		hexHash := hex.EncodeToString(entry.Hash)
		m[hexHash] = entry
	}

	return m
}
