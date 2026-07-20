package box_format

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
)

type CheckedOutHeaderDeleted struct {
	mad_domain_interfaces.ConfigDryRunGetter
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
