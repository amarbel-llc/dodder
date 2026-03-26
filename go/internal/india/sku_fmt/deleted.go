package sku_fmt

import (
	"code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/fields"
	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type itemDeletedStringFormatWriter struct {
	domain_interfaces.Config
	rightAlignedWriter   interfaces.StringEncoderTo[string]
	idStringFormatWriter interfaces.StringEncoderTo[string]
	fieldsFormatWriter   interfaces.StringEncoderTo[string_format_writer.Box]
}

func MakeItemDeletedStringWriterFormat(
	config domain_interfaces.Config,
	co string_format_writer.ColorOptions,
	fieldsFormatWriter interfaces.StringEncoderTo[string_format_writer.Box],
) *itemDeletedStringFormatWriter {
	return &itemDeletedStringFormatWriter{
		Config:             config,
		rightAlignedWriter: string_format_writer.MakeRightAligned(),
		idStringFormatWriter: string_format_writer.MakeColor(
			co,
			string_format_writer.MakeString[string](),
			fields.TypeId,
		),
		fieldsFormatWriter: fieldsFormatWriter,
	}
}

func (f *itemDeletedStringFormatWriter) EncodeStringTo(
	object *sku.Transacted,
	stringWriter interfaces.WriterAndStringWriter,
) (n int64, err error) {
	var (
		n1 int
		n2 int64
	)

	prefixOne := string_format_writer.StringDeleted

	if f.IsDryRun() {
		prefixOne = string_format_writer.StringWouldDelete
	}

	n2, err = f.rightAlignedWriter.EncodeStringTo(prefixOne, stringWriter)
	n += n2

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n1, err = stringWriter.WriteString("[")
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	var box string_format_writer.Box
	for field := range object.GetMetadata().GetIndex().GetFields() {
		box.Contents.Append(string_format_writer.FormattedField{Field: field})
	}

	n2, err = f.fieldsFormatWriter.EncodeStringTo(
		box,
		stringWriter,
	)
	n += n2

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	n1, err = stringWriter.WriteString("]")
	n += int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
