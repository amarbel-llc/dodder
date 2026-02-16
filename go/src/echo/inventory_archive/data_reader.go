package inventory_archive

import (
	"bytes"
	"encoding/binary"
	"io"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
)

type DataReader struct {
	reader          io.ReadSeeker
	hashFormatId    string
	compressionType compression_type.CompressionType
	hashSize        int
	dataStart       int64
}

func NewDataReader(
	r io.ReadSeeker,
) (dr *DataReader, err error) {
	dr = &DataReader{
		reader: r,
	}

	if err = dr.readHeader(); err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	return dr, nil
}

func (dr *DataReader) readHeader() (err error) {
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

	if version != DataFileVersion {
		err = errors.Errorf(
			"unsupported version: got %d, want %d",
			version,
			DataFileVersion,
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

	// compression: 1 byte
	var compressionByte [1]byte

	if _, err = io.ReadFull(dr.reader, compressionByte[:]); err != nil {
		err = errors.Wrapf(err, "reading compression byte")
		return err
	}

	dr.compressionType, err = ByteToCompression(compressionByte[0])
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	// flags: 2 bytes reserved
	var flags [2]byte

	if _, err = io.ReadFull(dr.reader, flags[:]); err != nil {
		err = errors.Wrapf(err, "reading flags")
		return err
	}

	dr.dataStart = int64(
		4 + // magic
			2 + // version
			1 + // hash_format_id_len
			len(hashFormatIdBytes) + // hash_format_id
			1 + // compression
			2, // flags
	)

	return nil
}

func (dr *DataReader) HashFormatId() string {
	return dr.hashFormatId
}

func (dr *DataReader) CompressionType() compression_type.CompressionType {
	return dr.compressionType
}

func (dr *DataReader) ReadEntry() (entry DataEntry, err error) {
	// Record current offset
	currentPos, err := dr.reader.Seek(0, io.SeekCurrent)
	if err != nil {
		err = errors.Wrapf(err, "getting current position")
		return entry, err
	}

	entry.Offset = uint64(currentPos)

	// Read hash
	entry.Hash = make([]byte, dr.hashSize)

	if _, err = io.ReadFull(dr.reader, entry.Hash); err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return entry, io.EOF
		}

		err = errors.Wrapf(err, "reading entry hash")
		return entry, err
	}

	// Read uncompressed_size
	if err = binary.Read(
		dr.reader,
		binary.BigEndian,
		&entry.UncompressedSize,
	); err != nil {
		err = errors.Wrapf(err, "reading uncompressed size")
		return entry, err
	}

	// Read compressed_size
	if err = binary.Read(
		dr.reader,
		binary.BigEndian,
		&entry.CompressedSize,
	); err != nil {
		err = errors.Wrapf(err, "reading compressed size")
		return entry, err
	}

	// Read compressed data
	compressedData := make([]byte, entry.CompressedSize)

	if _, err = io.ReadFull(dr.reader, compressedData); err != nil {
		err = errors.Wrapf(err, "reading compressed data")
		return entry, err
	}

	// Decompress data
	decompressReader, err := dr.compressionType.WrapReader(
		bytes.NewReader(compressedData),
	)
	if err != nil {
		err = errors.Wrapf(err, "creating decompression reader")
		return entry, err
	}

	entry.Data, err = io.ReadAll(decompressReader)
	if err != nil {
		err = errors.Wrapf(err, "decompressing data")
		return entry, err
	}

	if err = decompressReader.Close(); err != nil {
		err = errors.Wrapf(err, "closing decompression reader")
		return entry, err
	}

	return entry, nil
}

func (dr *DataReader) ReadAllEntries() (entries []DataEntry, err error) {
	// Seek to the start of entries
	if _, err = dr.reader.Seek(dr.dataStart, io.SeekStart); err != nil {
		err = errors.Wrapf(err, "seeking to data start")
		return nil, err
	}

	// First, determine the total size to know where the footer starts
	totalSize, err := dr.reader.Seek(0, io.SeekEnd)
	if err != nil {
		err = errors.Wrapf(err, "seeking to end")
		return nil, err
	}

	// Footer is: entry_count (8 bytes) + checksum (hashSize bytes)
	footerSize := int64(8 + dr.hashSize)
	entriesEnd := totalSize - footerSize

	// Seek back to data start
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

func (dr *DataReader) ReadEntryAt(
	offset uint64,
) (entry DataEntry, err error) {
	if _, err = dr.reader.Seek(int64(offset), io.SeekStart); err != nil {
		err = errors.Wrapf(err, "seeking to offset %d", offset)
		return entry, err
	}

	return dr.ReadEntry()
}

func (dr *DataReader) Validate() (err error) {
	// Seek to end to get total size
	totalSize, err := dr.reader.Seek(0, io.SeekEnd)
	if err != nil {
		err = errors.Wrapf(err, "seeking to end for validation")
		return err
	}

	// Footer: entry_count (8 bytes) + checksum (hashSize bytes)
	footerSize := int64(8 + dr.hashSize)

	if totalSize < footerSize {
		err = errors.Errorf("file too small for footer: %d bytes", totalSize)
		return err
	}

	// Read the stored checksum from the end
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

	// Compute checksum of everything before the checksum
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
