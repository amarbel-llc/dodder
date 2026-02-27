package object_metadata_fmt_triple_hyphen

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/charlie/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/golf/objects"
)

type (
	Formatter interface {
		FormatMetadata(io.Writer, FormatterContext) (int64, error)
	}

	Parser interface {
		ParseMetadata(io.Reader, ParserContext) (int64, error)
	}

	FormatterOptions = checkout_options.TextFormatterOptions

	// TODO make a reliable constructor for this
	FormatterContext struct {
		FormatterOptions
		objects.EncoderContext
	}

	ParserContext interface {
		objects.DecoderContext
	}

	FormatterFamily struct {
		BlobPath     Formatter
		InlineBlob   Formatter
		MetadataOnly Formatter
		BlobOnly     Formatter
	}

	Format struct {
		FormatterFamily
		Parser
	}

	funcWrite = interfaces.FuncWriterElementInterface[FormatterContext]
)
