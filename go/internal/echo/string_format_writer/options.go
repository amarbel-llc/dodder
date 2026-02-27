package string_format_writer

type (
	ColorType string

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
