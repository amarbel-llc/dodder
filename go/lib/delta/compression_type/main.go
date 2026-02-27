package compression_type

import (
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/charlie/ohio"
	"github.com/DataDog/zstd"
)

type ErrUnsupportedCompression string

func (err ErrUnsupportedCompression) Error() string {
	return fmt.Sprintf("unsupported compression type: %q", string(err))
}

func (ErrUnsupportedCompression) Is(target error) (ok bool) {
	_, ok = target.(ErrUnsupportedCompression)
	return ok
}

const (
	CompressionTypeEmpty = CompressionType("")
	CompressionTypeNone  = CompressionType("none")
	CompressionTypeGzip  = CompressionType("gzip")
	CompressionTypeZlib  = CompressionType("zlib")
	CompressionTypeZstd  = CompressionType("zstd")

	CompressionTypeDefault = CompressionTypeZstd
)

type CompressionType string

var _ interfaces.CommandComponentWriter = (*CompressionType)(nil)

func (compressionType *CompressionType) GetBlobCompression() interfaces.CLIFlagIOWrapper {
	return compressionType
}

func (compressionType *CompressionType) SetFlagDefinitions(
	flagDefinitions interfaces.CLIFlagDefinitions,
) {
	flagDefinitions.Var(compressionType, "compression-type", "")
}

func (compressionType CompressionType) GetCLICompletion() map[string]string {
	return map[string]string{
		CompressionTypeNone.String(): "",
		CompressionTypeGzip.String(): "",
		CompressionTypeZlib.String(): "",
		CompressionTypeZstd.String(): "",
	}
}

func (compressionType CompressionType) String() string {
	return string(compressionType)
}

func (compressionType *CompressionType) Set(value string) (err error) {
	valueClean := CompressionType(strings.TrimSpace(strings.ToLower(value)))

	switch valueClean {
	case CompressionTypeGzip,
		CompressionTypeNone,
		CompressionTypeEmpty,
		CompressionTypeZstd,
		CompressionTypeZlib:
		*compressionType = valueClean

	default:
		err = ErrUnsupportedCompression(value)
	}

	return err
}

func (compressionType CompressionType) WrapReader(
	readerIn io.Reader,
) (readerOut io.ReadCloser, err error) {
	switch compressionType {
	case CompressionTypeGzip:
		readerOut, err = gzip.NewReader(readerIn)

	case CompressionTypeZlib:
		readerOut, err = zlib.NewReader(readerIn)

	case CompressionTypeZstd:
		readerOut = zstd.NewReader(readerIn)

	default:
		readerOut = ohio.NopCloser(readerIn)
	}

	return readerOut, err
}

func (compressionType CompressionType) WrapWriter(
	writerIn io.Writer,
) (io.WriteCloser, error) {
	switch compressionType {
	case CompressionTypeGzip:
		return gzip.NewWriter(writerIn), nil

	case CompressionTypeZlib:
		return zlib.NewWriter(writerIn), nil

	case CompressionTypeZstd:
		return zstd.NewWriter(writerIn), nil

	default:
		return ohio.NopWriteCloser(writerIn), nil
	}
}
