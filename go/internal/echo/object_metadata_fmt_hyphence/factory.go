package object_metadata_fmt_hyphence

import (
	"code.linenisgreat.com/dodder/go/lib/bravo/script_config"
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	mad_env_dir "code.linenisgreat.com/madder/go/pkgs/env_dir"
)

type Factory struct {
	EnvDir        mad_env_dir.Env
	BlobStore     mad_domain_interfaces.BlobStore
	BlobFormatter script_config.RemoteScript
	BlobTreeDir   string

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

func (factory Factory) getBlobDigestType() mad_domain_interfaces.FormatHash {
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
		formatterComponents.writeReferencedObjects,
		formatterComponents.writeBlobReferences,
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
		formatterComponents.writeReferencedObjects,
		formatterComponents.writeBlobReferences,
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
		formatterComponents.writeReferencedObjects,
		formatterComponents.writeBlobReferences,
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
