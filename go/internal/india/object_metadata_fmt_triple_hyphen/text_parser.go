package object_metadata_fmt_triple_hyphen

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/script_config"
	"code.linenisgreat.com/dodder/go/internal/delta/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/triple_hyphen_io"
	"code.linenisgreat.com/dodder/go/internal/golf/objects"
)

type textParser struct {
	hashType      domain_interfaces.FormatHash
	blobWriter    domain_interfaces.BlobWriterFactory
	blobFormatter script_config.RemoteScript
}

func (parser textParser) ParseMetadata(
	reader io.Reader,
	context ParserContext,
) (n int64, err error) {
	metadata := context.GetMetadataMutable()
	objects.Resetter.Reset(metadata)

	var n1 int64

	parser2 := &textParser2{
		BlobWriterFactory: parser.blobWriter,
		hashType:          parser.hashType,
		ParserContext:     context,
	}

	var blobWriter domain_interfaces.BlobWriter

	if blobWriter, err = parser.blobWriter.MakeBlobWriter(nil); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	if blobWriter == nil {
		err = errors.ErrorWithStackf("blob writer is nil")
		return n, err
	}

	defer errors.DeferredCloser(&err, blobWriter)

	metadataReader := triple_hyphen_io.Reader{
		Metadata: parser2,
		Blob:     blobWriter,
	}

	if n, err = metadataReader.ReadFrom(reader); err != nil {
		n += n1
		err = errors.Wrap(err)
		return n, err
	}

	n += n1

	inlineBlobDigest := blobWriter.GetMarklId()

	if !metadata.GetBlobDigest().IsNull() && !parser2.Blob.GetDigest().IsNull() {
		err = errors.Wrap(
			MakeErrHasInlineBlobAndFilePath(
				&parser2.Blob,
				inlineBlobDigest,
			),
		)

		return n, err
	} else if !parser2.Blob.GetDigest().IsNull() {
		metadata.GetIndexMutable().GetFieldsMutable().Append(
			string_format_writer.Field{
				Key:       "blob",
				Value:     parser2.Blob.GetPath(),
				ColorType: string_format_writer.ColorTypeId,
			},
		)

		metadata.GetBlobDigestMutable().ResetWithMarklId(parser2.Blob.GetDigest())
	}

	switch {
	case metadata.GetBlobDigest().IsNull() && !inlineBlobDigest.IsNull():
		metadata.GetBlobDigestMutable().ResetWithMarklId(inlineBlobDigest)

	case !metadata.GetBlobDigest().IsNull() && inlineBlobDigest.IsNull():
		// noop

	case !metadata.GetBlobDigest().IsNull() && !inlineBlobDigest.IsNull() &&
		!markl.Equals(metadata.GetBlobDigest(), inlineBlobDigest):
		err = errors.Wrap(
			MakeErrHasInlineBlobAndMetadataBlobId(
				inlineBlobDigest,
				metadata.GetBlobDigest(),
			),
		)

		return n, err
	}

	return n, err
}
