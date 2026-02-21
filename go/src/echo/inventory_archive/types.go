package inventory_archive

import (
	"crypto/sha256"
	"crypto/sha512"
	"hash"

	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/charlie/compression_type"
	"golang.org/x/crypto/blake2b"
)

const (
	DataFileMagic  = "DIAR"
	IndexFileMagic = "DIAX"
	CacheFileMagic = "DIAC"

	DataFileVersion  uint16 = 0
	IndexFileVersion uint16 = 0
	CacheFileVersion uint16 = 0

	DataFileExtension  = ".inventory_archive-v0"
	IndexFileExtension = ".inventory_archive_index-v0"
	CacheFileName      = "index_cache-v0"

	CompressionByteNone byte = 0
	CompressionByteGzip byte = 1
	CompressionByteZlib byte = 2
	CompressionByteZstd byte = 3
)

const (
	DataFileVersionV1  uint16 = 1
	IndexFileVersionV1 uint16 = 1
	CacheFileVersionV1 uint16 = 1

	DataFileExtensionV1  = ".inventory_archive-v1"
	IndexFileExtensionV1 = ".inventory_archive_index-v1"
	CacheFileNameV1      = "index_cache-v1"

	EntryTypeFull  byte = 0x00
	EntryTypeDelta byte = 0x01

	FlagHasDeltas         uint16 = 1 << 0
	FlagReservedCrossArch uint16 = 1 << 1
)

type DataEntry struct {
	Hash             []byte
	UncompressedSize uint64
	CompressedSize   uint64
	Data             []byte
	Offset           uint64
}

type IndexEntry struct {
	Hash           []byte
	PackOffset     uint64
	CompressedSize uint64
}

type CacheEntry struct {
	Hash            []byte
	ArchiveChecksum []byte
	Offset          uint64
	CompressedSize  uint64
}

type DataEntryV1 struct {
	Hash             []byte
	EntryType        byte
	Encoding         byte
	UncompressedSize uint64
	CompressedSize   uint64
	Data             []byte
	Offset           uint64
	// Delta-specific fields (only set when EntryType == EntryTypeDelta)
	DeltaAlgorithm byte
	BaseHash       []byte
}

type IndexEntryV1 struct {
	Hash           []byte
	PackOffset     uint64
	CompressedSize uint64
	EntryType      byte
	BaseOffset     uint64
}

type CacheEntryV1 struct {
	Hash            []byte
	ArchiveChecksum []byte
	Offset          uint64
	CompressedSize  uint64
	EntryType       byte
	BaseOffset      uint64
}

var compressionToByteMap = map[compression_type.CompressionType]byte{
	compression_type.CompressionTypeNone:  CompressionByteNone,
	compression_type.CompressionTypeEmpty: CompressionByteNone,
	compression_type.CompressionTypeGzip:  CompressionByteGzip,
	compression_type.CompressionTypeZlib:  CompressionByteZlib,
	compression_type.CompressionTypeZstd:  CompressionByteZstd,
}

var byteToCompressionMap = map[byte]compression_type.CompressionType{
	CompressionByteNone: compression_type.CompressionTypeNone,
	CompressionByteGzip: compression_type.CompressionTypeGzip,
	CompressionByteZlib: compression_type.CompressionTypeZlib,
	CompressionByteZstd: compression_type.CompressionTypeZstd,
}

func CompressionToByte(
	ct compression_type.CompressionType,
) (b byte, err error) {
	var ok bool

	if b, ok = compressionToByteMap[ct]; !ok {
		err = errors.Errorf("unsupported compression type: %q", ct)
	}

	return b, err
}

func ByteToCompression(
	b byte,
) (ct compression_type.CompressionType, err error) {
	var ok bool

	if ct, ok = byteToCompressionMap[b]; !ok {
		err = errors.Errorf("unsupported compression byte: %d", b)
	}

	return ct, err
}

type hashConstructor func() hash.Hash

var hashConstructors = map[string]hashConstructor{
	"sha256": sha256.New,
	"sha512": sha512.New,
	"blake2b256": func() hash.Hash {
		h, _ := blake2b.New256(nil)
		return h
	},
	"blake2b512": func() hash.Hash {
		h, _ := blake2b.New512(nil)
		return h
	},
}

var hashSizes = map[string]int{
	"sha256":     sha256.Size,
	"sha512":     sha512.Size,
	"blake2b256": blake2b.Size256,
	"blake2b512": blake2b.Size,
}

func newHashForFormat(formatId string) (hash.Hash, error) {
	constructor, ok := hashConstructors[formatId]
	if !ok {
		return nil, errors.Errorf("unsupported hash format: %q", formatId)
	}

	return constructor(), nil
}

func hashSizeForFormat(formatId string) (int, error) {
	size, ok := hashSizes[formatId]
	if !ok {
		return 0, errors.Errorf("unsupported hash format: %q", formatId)
	}

	return size, nil
}
