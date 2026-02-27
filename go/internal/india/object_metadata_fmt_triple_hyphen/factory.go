package object_metadata_fmt_triple_hyphen

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/charlie/script_config"
	"code.linenisgreat.com/dodder/go/internal/hotel/env_dir"
)

type Factory struct {
	EnvDir        env_dir.Env
	BlobStore     domain_interfaces.BlobStore
	BlobFormatter script_config.RemoteScript

	AllowMissingTypeSig bool
}

func (factory Factory) Make() Format {
	return Format{
		Parser:          factory.MakeTextParser(),
		FormatterFamily: factory.MakeFormatterFamily(),
	}
}

func (factory Factory) MakeFormatterFamily() FormatterFamily {
	return FormatterFamily{
		BlobPath:     factory.makeFormatterMetadataBlobPath(),
		InlineBlob:   factory.makeFormatterMetadataInlineBlob(),
		MetadataOnly: factory.makeFormatterMetadataOnly(),
		BlobOnly:     factory.makeFormatterExcludeMetadata(),
	}
}

func (factory Factory) MakeTextParser() Parser {
	if factory.BlobStore == nil {
		panic("nil BlobWriterFactory")
	}

	return textParser{
		hashType:      factory.getBlobDigestType(),
		blobWriter:    factory.BlobStore,
		blobFormatter: factory.BlobFormatter,
	}
}

func (factory Factory) getBlobDigestType() domain_interfaces.FormatHash {
	hashType := factory.BlobStore.GetDefaultHashType()

	if hashType == nil {
		panic("no hash type set")
	}

	return hashType
}

func (factory Factory) makeFormatterMetadataBlobPath() formatter {
	formatterComponents := formatterComponents(factory)

	return formatter{
		formatterComponents.writeBoundary,
		formatterComponents.writeCommonMetadataFormat,
		formatterComponents.writeBlobPath,
		formatterComponents.getWriteTypeAndSigFunc(),
		formatterComponents.writeComments,
		formatterComponents.writeBoundary,
	}
}

func (factory Factory) makeFormatterMetadataOnly() formatter {
	formatterComponents := formatterComponents(factory)

	return formatter{
		formatterComponents.writeBoundary,
		formatterComponents.writeCommonMetadataFormat,
		formatterComponents.writeBlobDigest,
		formatterComponents.getWriteTypeAndSigFunc(),
		formatterComponents.writeComments,
		formatterComponents.writeBoundary,
	}
}

func (factory Factory) makeFormatterMetadataInlineBlob() formatter {
	formatterComponents := formatterComponents(factory)

	return formatter{
		formatterComponents.writeBoundary,
		formatterComponents.writeCommonMetadataFormat,
		formatterComponents.getWriteTypeAndSigFunc(),
		formatterComponents.writeComments,
		formatterComponents.writeBoundary,
		formatterComponents.writeNewLine,
		formatterComponents.writeBlob,
	}
}

func (factory Factory) makeFormatterExcludeMetadata() formatter {
	formatterComponents := formatterComponents(factory)

	return formatter{
		formatterComponents.writeBlob,
	}
}
