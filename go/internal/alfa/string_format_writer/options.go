package string_format_writer

import "code.linenisgreat.com/dodder/go/internal/alfa/fields"

type (
	ColorType = fields.Type

	ColorOptions struct {
		OffEntirely bool
	}

	OutputOptions struct {
		ColorOptionsOut ColorOptions
		ColorOptionsErr ColorOptions
	}
)

func (co ColorOptions) SetOffEntirely(v bool) ColorOptions {
	co.OffEntirely = v
	return co
}
