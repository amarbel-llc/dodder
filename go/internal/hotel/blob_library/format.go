package blob_library

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

type format[
	BLOB any,
	BLOB_PTR interfaces.Ptr[BLOB],
] struct {
	interfaces.DecoderFromReader[BLOB_PTR]
	domain_interfaces.SavedBlobFormatter
	interfaces.EncoderToWriter[BLOB_PTR]
}

func MakeBlobFormat[
	BLOB any,
	BLOB_PTR interfaces.Ptr[BLOB],
](
	decoder interfaces.DecoderFromReader[BLOB_PTR],
	encoder interfaces.EncoderToWriter[BLOB_PTR],
	blobReader domain_interfaces.BlobReaderFactory,
) domain_interfaces.Format[BLOB, BLOB_PTR] {
	return format[BLOB, BLOB_PTR]{
		DecoderFromReader:  decoder,
		EncoderToWriter:    encoder,
		SavedBlobFormatter: MakeSavedBlobFormatter(blobReader),
	}
}

func (af format[BLOB, BLOB_PTR]) EncodeTo(
	object BLOB_PTR,
	writer io.Writer,
) (n int64, err error) {
	if af.EncoderToWriter == nil {
		err = errors.ErrorWithStackf("no ParsedBlobFormatter")
	} else {
		n, err = af.EncoderToWriter.EncodeTo(object, writer)
	}

	return n, err
}
