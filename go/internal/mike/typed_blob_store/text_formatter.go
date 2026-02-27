package typed_blob_store

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/bravo/checkout_mode"
	"code.linenisgreat.com/dodder/go/internal/charlie/checkout_options"
	"code.linenisgreat.com/dodder/go/lib/charlie/script_config"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/india/object_metadata_fmt_triple_hyphen"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_repo"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
)

func MakeTextFormatter(
	envRepo env_repo.Env,
	options checkout_options.TextFormatterOptions,
	inlineTypeChecker ids.InlineTypeChecker,
	checkoutMode checkout_mode.Mode,
) textFormatter {
	return MakeTextFormatterWithBlobFormatter(
		envRepo,
		options,
		inlineTypeChecker,
		nil,
		checkoutMode,
	)
}

func MakeTextFormatterWithBlobFormatter(
	envRepo env_repo.Env,
	options checkout_options.TextFormatterOptions,
	inlineTypeChecker ids.InlineTypeChecker,
	formatter script_config.RemoteScript,
	checkoutMode checkout_mode.Mode,
) textFormatter {
	return textFormatter{
		options:           options,
		InlineTypeChecker: inlineTypeChecker,
		FormatterFamily: object_metadata_fmt_triple_hyphen.Factory{
			EnvDir:        envRepo,
			BlobStore:     envRepo.GetDefaultBlobStore(),
			BlobFormatter: formatter,
		}.MakeFormatterFamily(),
		checkoutMode: checkoutMode,
	}
}

type textFormatter struct {
	ids.InlineTypeChecker
	options checkout_options.TextFormatterOptions
	object_metadata_fmt_triple_hyphen.FormatterFamily
	checkoutMode checkout_mode.Mode
}

func (formatter textFormatter) EncodeStringTo(
	object *sku.Transacted,
	writer io.Writer,
) (n int64, err error) {
	context := object_metadata_fmt_triple_hyphen.FormatterContext{
		EncoderContext:   object,
		FormatterOptions: formatter.options,
	}

	switch {
	case formatter.checkoutMode.IsMetadataOnly():
		n, err = formatter.MetadataOnly.FormatMetadata(writer, context)

	default:
		if object.GetGenre().IsConfig() {
			n, err = formatter.BlobOnly.FormatMetadata(writer, context)
		} else if formatter.InlineTypeChecker.IsInlineType(object.GetType()) {
			n, err = formatter.InlineBlob.FormatMetadata(writer, context)
		} else {
			n, err = formatter.MetadataOnly.FormatMetadata(writer, context)
		}
	}

	return n, err
}

func (tf textFormatter) WriteStringFormatWithMode(
	writer io.Writer,
	object *sku.Transacted,
	mode checkout_mode.Mode,
) (n int64, err error) {
	ctx := object_metadata_fmt_triple_hyphen.FormatterContext{
		EncoderContext:   object,
		FormatterOptions: tf.options,
	}

	if object.GetGenre().IsConfig() ||
		mode.IsBlobOnly() {
		n, err = tf.BlobOnly.FormatMetadata(writer, ctx)
	} else if tf.InlineTypeChecker.IsInlineType(object.GetType()) {
		n, err = tf.InlineBlob.FormatMetadata(writer, ctx)
	} else {
		n, err = tf.MetadataOnly.FormatMetadata(writer, ctx)
	}

	return n, err
}
