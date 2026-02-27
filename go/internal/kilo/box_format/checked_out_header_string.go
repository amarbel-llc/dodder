package box_format

import (
	"code.linenisgreat.com/dodder/go/internal/delta/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
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
