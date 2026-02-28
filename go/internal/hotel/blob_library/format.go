package blob_library

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
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
