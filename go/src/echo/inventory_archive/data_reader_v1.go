package inventory_archive

import (
	"bytes"
	"encoding/binary"
	"io"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
)

type DataReaderV1 struct {
	reader          io.ReadSeeker
	hashFormatId    string
	compressionType compression_type.CompressionType
	hashSize        int
	flags           uint16
	dataStart       int64
}

func NewDataReaderV1(
	r io.ReadSeeker,
) (dr *DataReaderV1, err error) {
	dr = &DataReaderV1{
		reader: r,
	}

	if err = dr.readHeader(); err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	return dr, nil
}

func (dr *DataReaderV1) readHeader() (err error) {
	// magic: 4 bytes
	magic := make([]byte, 4)

	if _, err = io.ReadFull(dr.reader, magic); err != nil {
		err = errors.Wrapf(err, "reading magic")
		return err
	}

	if string(magic) != DataFileMagic {
		err = errors.Errorf(
			"invalid magic: got %q, want %q",
			string(magic),
			DataFileMagic,
		)
		return err
	}

	// version: 2 bytes uint16 BigEndian
	var version uint16

	if err = binary.Read(
		dr.reader,
		binary.BigEndian,
		&version,
	); err != nil {
		err = errors.Wrapf(err, "reading version")
		return err
	}

	if version != DataFileVersionV1 {
		err = errors.Errorf(
			"unsupported version: got %d, want %d",
			version,
			DataFileVersionV1,
		)
		return err
	}

	// hash_format_id_len: 1 byte
	var hashFormatIdLen [1]byte

	if _, err = io.ReadFull(dr.reader, hashFormatIdLen[:]); err != nil {
		err = errors.Wrapf(err, "reading hash format id length")
		return err
	}

	// hash_format_id: variable
	hashFormatIdBytes := make([]byte, hashFormatIdLen[0])

	if _, err = io.ReadFull(dr.reader, hashFormatIdBytes); err != nil {
		err = errors.Wrapf(err, "reading hash format id")
		return err
	}

	dr.hashFormatId = string(hashFormatIdBytes)

	dr.hashSize, err = hashSizeForFormat(dr.hashFormatId)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	// default_encoding: 1 byte
	var compressionByte [1]byte

	if _, err = io.ReadFull(dr.reader, compressionByte[:]); err != nil {
		err = errors.Wrapf(err, "reading default encoding byte")
		return err
	}

	dr.compressionType, err = ByteToCompression(compressionByte[0])
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	// flags: 2 bytes
	if err = binary.Read(
		dr.reader,
		binary.BigEndian,
		&dr.flags,
	); err != nil {
		err = errors.Wrapf(err, "reading flags")
		return err
	}

	dr.dataStart = int64(
		4 + // magic
			2 + // version
			1 + // hash_format_id_len
			len(hashFormatIdBytes) + // hash_format_id
			1 + // default_encoding
			2, // flags
	)

	return nil
}

func (dr *DataReaderV1) HashFormatId() string {
	return dr.hashFormatId
}

func (dr *DataReaderV1) CompressionType() compression_type.CompressionType {
	return dr.compressionType
}

func (dr *DataReaderV1) Flags() uint16 {
	return dr.flags
}

func (dr *DataReaderV1) ReadEntry() (entry DataEntryV1, err error) {
	currentPos, err := dr.reader.Seek(0, io.SeekCurrent)
	if err != nil {
		err = errors.Wrapf(err, "getting current position")
		return entry, err
	}

	entry.Offset = uint64(currentPos)

	// hash
	entry.Hash = make([]byte, dr.hashSize)

	if _, err = io.ReadFull(dr.reader, entry.Hash); err != nil {
		if err == io.EOF {
			return entry, io.EOF
		}

		err = errors.Wrapf(err, "reading entry hash")
		return entry, err
	}

	// entry_type
	var entryTypeByte [1]byte

	if _, err = io.ReadFull(dr.reader, entryTypeByte[:]); err != nil {
		err = errors.Wrapf(err, "reading entry type")
		return entry, err
	}

	entry.EntryType = entryTypeByte[0]

	// encoding
	var encodingByte [1]byte

	if _, err = io.ReadFull(dr.reader, encodingByte[:]); err != nil {
		err = errors.Wrapf(err, "reading encoding")
		return entry, err
	}

	entry.Encoding = encodingByte[0]

	entryCompression, err := ByteToCompression(entry.Encoding)
	if err != nil {
		err = errors.Wrap(err)
		return entry, err
	}

	switch entry.EntryType {
	case EntryTypeFull:
		err = dr.readFullEntryBody(&entry, entryCompression)

	case EntryTypeDelta:
		err = dr.readDeltaEntryBody(&entry, entryCompression)

	default:
		err = errors.Errorf("unknown entry type: %d", entry.EntryType)
	}

	if err != nil {
		return entry, err
	}

	return entry, nil
}

func (dr *DataReaderV1) readFullEntryBody(
	entry *DataEntryV1,
	entryCompression compression_type.CompressionType,
) (err error) {
	// uncompressed_size
	if err = binary.Read(
		dr.reader,
		binary.BigEndian,
		&entry.UncompressedSize,
	); err != nil {
		err = errors.Wrapf(err, "reading uncompressed size")
		return err
	}

	// compressed_size
	if err = binary.Read(
		dr.reader,
		binary.BigEndian,
		&entry.CompressedSize,
	); err != nil {
		err = errors.Wrapf(err, "reading compressed size")
		return err
	}

	// compressed_data
	compressedData := make([]byte, entry.CompressedSize)

	if _, err = io.ReadFull(dr.reader, compressedData); err != nil {
		err = errors.Wrapf(err, "reading compressed data")
		return err
	}

	// Decompress
	decompressReader, err := entryCompression.WrapReader(
		bytes.NewReader(compressedData),
	)
	if err != nil {
		err = errors.Wrapf(err, "creating decompression reader")
		return err
	}

	entry.Data, err = io.ReadAll(decompressReader)
	if err != nil {
		err = errors.Wrapf(err, "decompressing data")
		return err
	}

	if err = decompressReader.Close(); err != nil {
		err = errors.Wrapf(err, "closing decompression reader")
		return err
	}

	return nil
}

func (dr *DataReaderV1) readDeltaEntryBody(
	entry *DataEntryV1,
	entryCompression compression_type.CompressionType,
) (err error) {
	// delta_algorithm
	var deltaAlgByte [1]byte

	if _, err = io.ReadFull(dr.reader, deltaAlgByte[:]); err != nil {
		err = errors.Wrapf(err, "reading delta algorithm")
		return err
	}

	entry.DeltaAlgorithm = deltaAlgByte[0]

	// base_hash
	entry.BaseHash = make([]byte, dr.hashSize)

	if _, err = io.ReadFull(dr.reader, entry.BaseHash); err != nil {
		err = errors.Wrapf(err, "reading base hash")
		return err
	}

	// uncompressed_size
	if err = binary.Read(
		dr.reader,
		binary.BigEndian,
		&entry.UncompressedSize,
	); err != nil {
		err = errors.Wrapf(err, "reading uncompressed size")
		return err
	}

	// delta_size
	if err = binary.Read(
		dr.reader,
		binary.BigEndian,
		&entry.CompressedSize,
	); err != nil {
		err = errors.Wrapf(err, "reading delta size")
		return err
	}

	// delta_data (compressed)
	compressedDelta := make([]byte, entry.CompressedSize)

	if _, err = io.ReadFull(dr.reader, compressedDelta); err != nil {
		err = errors.Wrapf(err, "reading delta data")
		return err
	}

	// Decompress delta payload
	decompressReader, err := entryCompression.WrapReader(
		bytes.NewReader(compressedDelta),
	)
	if err != nil {
		err = errors.Wrapf(err, "creating decompression reader for delta")
		return err
	}

	entry.Data, err = io.ReadAll(decompressReader)
	if err != nil {
		err = errors.Wrapf(err, "decompressing delta payload")
		return err
	}

	if err = decompressReader.Close(); err != nil {
		err = errors.Wrapf(err, "closing decompression reader for delta")
		return err
	}

	return nil
}

func (dr *DataReaderV1) ReadAllEntries() (entries []DataEntryV1, err error) {
	if _, err = dr.reader.Seek(dr.dataStart, io.SeekStart); err != nil {
		err = errors.Wrapf(err, "seeking to data start")
		return nil, err
	}

	totalSize, err := dr.reader.Seek(0, io.SeekEnd)
	if err != nil {
		err = errors.Wrapf(err, "seeking to end")
		return nil, err
	}

	footerSize := int64(8 + dr.hashSize)
	entriesEnd := totalSize - footerSize

	if _, err = dr.reader.Seek(dr.dataStart, io.SeekStart); err != nil {
		err = errors.Wrapf(err, "seeking back to data start")
		return nil, err
	}

	for {
		currentPos, posErr := dr.reader.Seek(0, io.SeekCurrent)
		if posErr != nil {
			err = errors.Wrapf(posErr, "getting current position")
			return nil, err
		}

		if currentPos >= entriesEnd {
			break
		}

		entry, readErr := dr.ReadEntry()
		if readErr != nil {
			if readErr == io.EOF {
				break
			}

			err = errors.Wrap(readErr)
			return nil, err
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func (dr *DataReaderV1) ReadEntryAt(
	offset uint64,
) (entry DataEntryV1, err error) {
	if _, err = dr.reader.Seek(int64(offset), io.SeekStart); err != nil {
		err = errors.Wrapf(err, "seeking to offset %d", offset)
		return entry, err
	}

	return dr.ReadEntry()
}

func (dr *DataReaderV1) Validate() (err error) {
	totalSize, err := dr.reader.Seek(0, io.SeekEnd)
	if err != nil {
		err = errors.Wrapf(err, "seeking to end for validation")
		return err
	}

	footerSize := int64(8 + dr.hashSize)

	if totalSize < footerSize {
		err = errors.Errorf("file too small for footer: %d bytes", totalSize)
		return err
	}

	checksumOffset := totalSize - int64(dr.hashSize)

	if _, err = dr.reader.Seek(checksumOffset, io.SeekStart); err != nil {
		err = errors.Wrapf(err, "seeking to checksum")
		return err
	}

	storedChecksum := make([]byte, dr.hashSize)

	if _, err = io.ReadFull(dr.reader, storedChecksum); err != nil {
		err = errors.Wrapf(err, "reading stored checksum")
		return err
	}

	hasher, err := newHashForFormat(dr.hashFormatId)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	if _, err = dr.reader.Seek(0, io.SeekStart); err != nil {
		err = errors.Wrapf(err, "seeking to start for hashing")
		return err
	}

	if _, err = io.CopyN(hasher, dr.reader, checksumOffset); err != nil {
		err = errors.Wrapf(err, "hashing file content")
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
