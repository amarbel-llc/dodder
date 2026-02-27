package env_ui

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/charlie/options_print"
	"code.linenisgreat.com/dodder/go/internal/delta/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/fd"
)

func (env *env) FormatOutputOptions(
	printOptions options_print.Options,
) (o string_format_writer.OutputOptions) {
	o.ColorOptionsOut = env.FormatColorOptionsOut(printOptions)
	o.ColorOptionsErr = env.FormatColorOptionsErr(printOptions)
	return o
}

func (env *env) shouldUseColorOutput(
	printOptions options_print.Options,
	fd fd.Std,
) bool {
	if env.options.IgnoreTtyState {
		return printOptions.PrintColors
	} else {
		return fd.IsTty() && printOptions.PrintColors
	}
}

func (env *env) FormatColorOptionsOut(
	printOptions options_print.Options,
) (o string_format_writer.ColorOptions) {
	o.OffEntirely = !env.shouldUseColorOutput(printOptions, env.GetOut())
	return o
}

func (env *env) FormatColorOptionsErr(
	printOptions options_print.Options,
) (o string_format_writer.ColorOptions) {
	o.OffEntirely = !env.shouldUseColorOutput(printOptions, env.GetErr())
	return o
}

func (env *env) StringFormatWriterFields(
	truncate string_format_writer.CliFormatTruncation,
	co string_format_writer.ColorOptions,
) interfaces.StringEncoderTo[string_format_writer.Box] {
	return string_format_writer.MakeBoxStringEncoder(truncate, co)
}
