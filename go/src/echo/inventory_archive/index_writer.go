package inventory_archive

import (
	"bytes"
	"encoding/binary"
	"io"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
)

func WriteIndex(
	w io.Writer,
	hashFormatId string,
	entries []IndexEntry,
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

	if err = verifySorted(entries, hashSize); err != nil {
		return nil, err
	}

	multiWriter := io.MultiWriter(w, hasher)

	if err = writeIndexHeader(
		multiWriter,
		hashFormatId,
		uint64(len(entries)),
	); err != nil {
		return nil, err
	}

	if err = writeIndexFanOut(multiWriter, entries); err != nil {
		return nil, err
	}

	if err = writeIndexEntries(multiWriter, entries); err != nil {
		return nil, err
	}

	checksum = hasher.Sum(nil)

	if _, err = w.Write(checksum); err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	return checksum, nil
}

func verifySorted(entries []IndexEntry, hashSize int) error {
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

func writeIndexHeader(
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
		IndexFileVersion,
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

func writeIndexFanOut(
	w io.Writer,
	entries []IndexEntry,
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

func writeIndexEntries(
	w io.Writer,
	entries []IndexEntry,
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
	}

	return nil
}
