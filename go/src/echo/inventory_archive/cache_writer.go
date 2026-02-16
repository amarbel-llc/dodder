package inventory_archive

import (
	"bytes"
	"encoding/binary"
	"io"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

func WriteCache(
	w io.Writer,
	hashFormatId string,
	entries []CacheEntry,
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

	if err = verifyCacheSorted(entries, hashSize); err != nil {
		return nil, err
	}

	multiWriter := io.MultiWriter(w, hasher)

	if err = writeCacheHeader(
		multiWriter,
		hashFormatId,
		uint64(len(entries)),
	); err != nil {
		return nil, err
	}

	if err = writeCacheEntries(multiWriter, entries); err != nil {
		return nil, err
	}

	checksum = hasher.Sum(nil)

	if _, err = w.Write(checksum); err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	return checksum, nil
}

func verifyCacheSorted(entries []CacheEntry, hashSize int) error {
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

func writeCacheHeader(
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
		CacheFileVersion,
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

func writeCacheEntries(
	w io.Writer,
	entries []CacheEntry,
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
	}

	return nil
}
