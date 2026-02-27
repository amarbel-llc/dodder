package descriptions

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/delta/string_format_writer"
)

type formatCliStringer struct {
	truncate           string_format_writer.CliFormatTruncation
	stringFormatWriter interfaces.StringEncoderTo[string]
}

func MakeCliFormatStringer(
	truncate string_format_writer.CliFormatTruncation,
	co string_format_writer.ColorOptions,
	quote bool,
) *formatCliStringer {
	sfw := string_format_writer.MakeString[string]()

	if quote {
		sfw = string_format_writer.MakeQuotedString[string]()
	}

	return &formatCliStringer{
		truncate: truncate,
		stringFormatWriter: string_format_writer.MakeColor(
			co,
			sfw,
			string_format_writer.ColorTypeUserData,
		),
	}
}

func (f *formatCliStringer) EncodeStringTo(
	k interfaces.Stringer,
	w interfaces.WriterAndStringWriter,
) (n int64, err error) {
	v := k.String()

	// TODO format ellipsis as outside quotes and not styled
	if f.truncate == string_format_writer.CliFormatTruncation66CharEllipsis && len(v) > 66 {
		v = v[:66] + "…"
	}

	return f.stringFormatWriter.EncodeStringTo(v, w)
}
