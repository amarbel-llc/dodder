package inventory_archive

import (
	"bytes"
	"encoding/binary"
	"hash"
	"io"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/compression_type"
)

type DataWriterV1 struct {
	writer          io.Writer
	hasher          hash.Hash
	multiWriter     io.Writer
	hashFormatId    string
	compressionType compression_type.CompressionType
	encryption      interfaces.IOWrapper
	hashSize        int
	flags           uint16
	entries         []DataEntryV1
	offset          uint64
}

func NewDataWriterV1(
	w io.Writer,
	hashFormatId string,
	ct compression_type.CompressionType,
	flags uint16,
	encryption interfaces.IOWrapper,
) (dw *DataWriterV1, err error) {
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

	if encryption != nil {
		flags |= FlagHasEncryptionV1
	}

	multiWriter := io.MultiWriter(w, hasher)

	dw = &DataWriterV1{
		writer:          w,
		hasher:          hasher,
		multiWriter:     multiWriter,
		hashFormatId:    hashFormatId,
		compressionType: ct,
		encryption:      encryption,
		hashSize:        hashSize,
		flags:           flags,
	}

	if err = dw.writeHeader(); err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	return dw, nil
}

func (dw *DataWriterV1) writeHeader() (err error) {
	// magic: 4 bytes
	if _, err = dw.multiWriter.Write([]byte(DataFileMagic)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// version: 2 bytes uint16 BigEndian
	if err = binary.Write(
		dw.multiWriter,
		binary.BigEndian,
		DataFileVersionV1,
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

	// default_encoding: 1 byte
	compressionByte, err := CompressionToByte(dw.compressionType)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	if _, err = dw.multiWriter.Write([]byte{compressionByte}); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// flags: 2 bytes
	if err = binary.Write(
		dw.multiWriter,
		binary.BigEndian,
		dw.flags,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	dw.offset = uint64(
		4 + // magic
			2 + // version
			1 + // hash_format_id_len
			len(hashFormatIdBytes) + // hash_format_id
			1 + // default_encoding
			2, // flags
	)

	return nil
}

func (dw *DataWriterV1) WriteFullEntry(
	entryHash []byte,
	data []byte,
) (err error) {
	entryOffset := dw.offset

	encodingByte, err := CompressionToByte(dw.compressionType)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	// hash
	if _, err = dw.multiWriter.Write(entryHash); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// entry_type
	if _, err = dw.multiWriter.Write([]byte{EntryTypeFull}); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// encoding
	if _, err = dw.multiWriter.Write([]byte{encodingByte}); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// Compress data
	logicalSize := uint64(len(data))

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

	// Encrypt if configured
	storedData := compressedData
	if dw.encryption != nil {
		var encryptedBuf bytes.Buffer
		encryptWriter, encErr := dw.encryption.WrapWriter(&encryptedBuf)
		if encErr != nil {
			err = errors.Wrap(encErr)
			return err
		}
		if _, encErr = encryptWriter.Write(compressedData); encErr != nil {
			err = errors.Wrap(encErr)
			return err
		}
		if encErr = encryptWriter.Close(); encErr != nil {
			err = errors.Wrap(encErr)
			return err
		}
		storedData = encryptedBuf.Bytes()
	}

	storedSize := uint64(len(storedData))

	// logical_size
	if err = binary.Write(
		dw.multiWriter,
		binary.BigEndian,
		logicalSize,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// stored_size
	if err = binary.Write(
		dw.multiWriter,
		binary.BigEndian,
		storedSize,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// payload
	if _, err = dw.multiWriter.Write(storedData); err != nil {
		err = errors.Wrap(err)
		return err
	}

	entry := DataEntryV1{
		Hash:        make([]byte, len(entryHash)),
		EntryType:   EntryTypeFull,
		Encoding:    encodingByte,
		LogicalSize: logicalSize,
		StoredSize:  storedSize,
		Offset:      entryOffset,
	}

	copy(entry.Hash, entryHash)

	dw.entries = append(dw.entries, entry)

	dw.offset += uint64(len(entryHash)) + // hash
		1 + // entry_type
		1 + // encoding
		8 + // logical_size
		8 + // stored_size
		storedSize // payload

	return nil
}

func (dw *DataWriterV1) WriteDeltaEntry(
	entryHash []byte,
	deltaAlgorithm byte,
	baseHash []byte,
	logicalSize uint64,
	deltaPayload []byte,
) (err error) {
	entryOffset := dw.offset

	encodingByte, err := CompressionToByte(dw.compressionType)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	// hash
	if _, err = dw.multiWriter.Write(entryHash); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// entry_type
	if _, err = dw.multiWriter.Write([]byte{EntryTypeDelta}); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// encoding
	if _, err = dw.multiWriter.Write([]byte{encodingByte}); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// delta_algorithm
	if _, err = dw.multiWriter.Write([]byte{deltaAlgorithm}); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// base_hash
	if _, err = dw.multiWriter.Write(baseHash); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// Compress delta payload
	var compressedBuf bytes.Buffer

	compressWriter, err := dw.compressionType.WrapWriter(&compressedBuf)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	if _, err = compressWriter.Write(deltaPayload); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = compressWriter.Close(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	compressedData := compressedBuf.Bytes()

	// Encrypt if configured
	storedData := compressedData
	if dw.encryption != nil {
		var encryptedBuf bytes.Buffer
		encryptWriter, encErr := dw.encryption.WrapWriter(&encryptedBuf)
		if encErr != nil {
			err = errors.Wrap(encErr)
			return err
		}
		if _, encErr = encryptWriter.Write(compressedData); encErr != nil {
			err = errors.Wrap(encErr)
			return err
		}
		if encErr = encryptWriter.Close(); encErr != nil {
			err = errors.Wrap(encErr)
			return err
		}
		storedData = encryptedBuf.Bytes()
	}

	storedSize := uint64(len(storedData))

	// logical_size
	if err = binary.Write(
		dw.multiWriter,
		binary.BigEndian,
		logicalSize,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// stored_size
	if err = binary.Write(
		dw.multiWriter,
		binary.BigEndian,
		storedSize,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	// payload
	if _, err = dw.multiWriter.Write(storedData); err != nil {
		err = errors.Wrap(err)
		return err
	}

	entry := DataEntryV1{
		Hash:           make([]byte, len(entryHash)),
		EntryType:      EntryTypeDelta,
		Encoding:       encodingByte,
		DeltaAlgorithm: deltaAlgorithm,
		BaseHash:       make([]byte, len(baseHash)),
		LogicalSize:    logicalSize,
		StoredSize:     storedSize,
		Offset:         entryOffset,
	}

	copy(entry.Hash, entryHash)
	copy(entry.BaseHash, baseHash)

	dw.entries = append(dw.entries, entry)

	dw.offset += uint64(len(entryHash)) + // hash
		1 + // entry_type
		1 + // encoding
		1 + // delta_algorithm
		uint64(len(baseHash)) + // base_hash
		8 + // logical_size
		8 + // stored_size
		storedSize // payload

	return nil
}

func (dw *DataWriterV1) Close() (
	checksum []byte,
	entries []DataEntryV1,
	err error,
) {
	entryCount := uint64(len(dw.entries))

	if err = binary.Write(
		dw.multiWriter,
		binary.BigEndian,
		entryCount,
	); err != nil {
		err = errors.Wrap(err)
		return nil, nil, err
	}

	checksum = dw.hasher.Sum(nil)

	if _, err = dw.writer.Write(checksum); err != nil {
		err = errors.Wrap(err)
		return nil, nil, err
	}

	return checksum, dw.entries, nil
}
