package box_format

import (
	"code.linenisgreat.com/dodder/go/internal/echo/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
)

type TransactedHeaderUserTai struct{}

func (f TransactedHeaderUserTai) WriteBoxHeader(
	header *string_format_writer.BoxHeader,
	object *sku.Transacted,
) (err error) {
	tai := object.GetTai()
	header.RightAligned = true
	header.Value = tai.Format(string_format_writer.StringFormatDateTime)

	return err
}
