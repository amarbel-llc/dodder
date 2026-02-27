package box_format

import (
	"code.linenisgreat.com/dodder/go/internal/echo/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
)

type CheckedOutHeaderString string

func (f CheckedOutHeaderString) WriteBoxHeader(
	header *string_format_writer.BoxHeader,
	co *sku.CheckedOut,
) (err error) {
	header.RightAligned = true
	header.Value = string(f)

	return err
}
