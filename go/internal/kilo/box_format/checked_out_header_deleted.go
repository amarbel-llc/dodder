package box_format

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/delta/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
)

type CheckedOutHeaderDeleted struct {
	domain_interfaces.ConfigDryRunGetter
}

func (f CheckedOutHeaderDeleted) WriteBoxHeader(
	header *string_format_writer.BoxHeader,
	co *sku.CheckedOut,
) (err error) {
	header.RightAligned = true

	if f.IsDryRun() {
		header.Value = "would delete"
	} else {
		header.Value = "deleted"
	}

	return err
}
