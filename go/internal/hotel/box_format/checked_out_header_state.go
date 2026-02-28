package box_format

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/bravo/checked_out_state"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
)

type CheckedOutHeaderState struct{}

func (f CheckedOutHeaderState) WriteBoxHeader(
	header *string_format_writer.BoxHeader,
	checkedOut *sku.CheckedOut,
) (err error) {
	header.RightAligned = true

	state := checkedOut.GetState()
	stateString := state.String()

	switch state {
	case checked_out_state.CheckedOut:
		if objects.EqualerSansTai.Equals(
			checkedOut.GetSku().GetMetadata(),
			checkedOut.GetSkuExternal().GetSku().GetMetadata(),
		) {
			header.Value = string_format_writer.StringSame
		} else {
			header.Value = string_format_writer.StringChanged
		}

	default:
		header.Value = stateString
	}

	return err
}
