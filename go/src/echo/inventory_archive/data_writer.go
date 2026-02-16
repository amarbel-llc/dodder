package inventory_archive

import (
	"bytes"
	"encoding/binary"
	"hash"
	"io"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
)

type DataWriter struct {
	writer          io.Writer
	hasher          hash.Hash
	multiWriter     io.Writer
	hashFormatId    string
	compressionType compression_type.CompressionType
	hashSize        int
	entries         []DataEntry
	offset          uint64
}

func NewDataWriter(
	w io.Writer,
	hashFormatId string,
	ct compression_type.CompressionType,
) (dw *DataWriter, err error) {
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

	multiWriter := io.MultiWriter(w, hasher)

	dw = &DataWriter{
		writer:          w,
		hasher:          hasher,
		multiWriter:     multiWriter,
		hashFormatId:    hashFormatId,
		compressionType: ct,
		hashSize:        hashSize,
	}

	if err = dw.writeHeader(); err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	return dw, nil
}

func (dw *DataWriter) writeHeader() (err error) {
	// magic: 4 bytes
	if _, err = dw.multiWriter.Write([]byte(DataFileMagic)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// version: 2 bytes uint16 BigEndian
	if err = binary.Write(
		dw.multiWriter,
		binary.BigEndian,
		DataFileVersion,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// hash_format_id_len: 1 byte
	hashFormatIdBytes := []byte(dw.hashFormatId)

	if len(hashFormatIdBytes) > 255 {
		err = errors.Errorf(
			"hash format id too long: %d bytes",
			len(hashFormatIdBytes),
		)
		return err
	}

	if _, err = dw.multiWriter.Write(
		[]byte{byte(len(hashFormatIdBytes))},
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// hash_format_id: variable
	if _, err = dw.multiWriter.Write(hashFormatIdBytes); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// compression: 1 byte
	compressionByte, err := CompressionToByte(dw.compressionType)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	if _, err = dw.multiWriter.Write([]byte{compressionByte}); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// flags: 2 bytes reserved (zeros)
	if _, err = dw.multiWriter.Write([]byte{0, 0}); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// Track the offset after the header
	dw.offset = uint64(
		4 + // magic
			2 + // version
			1 + // hash_format_id_len
			len(hashFormatIdBytes) + // hash_format_id
			1 + // compression
			2, // flags
	)

	return nil
}

func (dw *DataWriter) WriteEntry(
	entryHash []byte,
	data []byte,
) (err error) {
	entryOffset := dw.offset

	// Write hash
	if _, err = dw.multiWriter.Write(entryHash); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// Write uncompressed_size
	uncompressedSize := uint64(len(data))

	if err = binary.Write(
		dw.multiWriter,
		binary.BigEndian,
		uncompressedSize,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// Compress data
	var compressedBuf bytes.Buffer

	compressWriter, err := dw.compressionType.WrapWriter(&compressedBuf)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	if _, err = compressWriter.Write(data); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = compressWriter.Close(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	compressedData := compressedBuf.Bytes()
	compressedSize := uint64(len(compressedData))

	// Write compressed_size
	if err = binary.Write(
		dw.multiWriter,
		binary.BigEndian,
		compressedSize,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// Write compressed data
	if _, err = dw.multiWriter.Write(compressedData); err != nil {
		err = errors.Wrap(err)
		return err
	}

	entry := DataEntry{
		Hash:             make([]byte, len(entryHash)),
		UncompressedSize: uncompressedSize,
		CompressedSize:   compressedSize,
		Offset:           entryOffset,
	}

	copy(entry.Hash, entryHash)

	dw.entries = append(dw.entries, entry)

	dw.offset += uint64(len(entryHash)) + // hash
		8 + // uncompressed_size
		8 + // compressed_size
		compressedSize // data

	return nil
}

func (dw *DataWriter) Close() (
	checksum []byte,
	entries []DataEntry,
	err error,
) {
	entryCount := uint64(len(dw.entries))

	// Write entry_count to both the output and the hasher
	if err = binary.Write(
		dw.multiWriter,
		binary.BigEndian,
		entryCount,
	); err != nil {
		err = errors.Wrap(err)
		return nil, nil, err
	}

	// Compute checksum of everything written so far
	checksum = dw.hasher.Sum(nil)

	// Write the checksum to the output only (not the hasher)
	if _, err = dw.writer.Write(checksum); err != nil {
		err = errors.Wrap(err)
		return nil, nil, err
	}

	return checksum, dw.entries, nil
}
