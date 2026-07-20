package env_ui

import (
	"code.linenisgreat.com/dodder/go/internal/0/options_print"
	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/madder/go/pkgs/fd"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

// These were Env methods until #151 bucket B Stage B follow-up.
// Moved to package-level functions so dodder env_ui.Env doesn't
// have to extend madder env_ui.Env with dodder-only methods —
// without that extension dodder's env_local can be aliased to
// madder's directly.

// FormatOutputOptions wraps FormatColorOptionsOut + Err for the
// common Both-streams case.
func FormatOutputOptions(
	env Env,
	printOptions options_print.Options,
) (o string_format_writer.OutputOptions) {
	o.ColorOptionsOut = FormatColorOptionsOut(env, printOptions)
	o.ColorOptionsErr = FormatColorOptionsErr(env, printOptions)
	return o
}

// FormatColorOptionsOut decides whether stdout should be coloured
// based on the env's TTY state and the caller's print options.
func FormatColorOptionsOut(
	env Env,
	printOptions options_print.Options,
) (o string_format_writer.ColorOptions) {
	o.OffEntirely = !shouldUseColorOutput(env, printOptions, env.GetOut())
	return o
}

// FormatColorOptionsErr decides whether stderr should be coloured.
func FormatColorOptionsErr(
	env Env,
	printOptions options_print.Options,
) (o string_format_writer.ColorOptions) {
	o.OffEntirely = !shouldUseColorOutput(env, printOptions, env.GetErr())
	return o
}

// StringFormatWriterFields builds dodder's box-formatted column
// writer with the truncation + color settings supplied. Free
// function (was previously a method) — has no env state of its
// own beyond the args.
func StringFormatWriterFields(
	truncate string_format_writer.CliFormatTruncation,
	co string_format_writer.ColorOptions,
) interfaces.StringEncoderTo[string_format_writer.Box] {
	return string_format_writer.MakeBoxStringEncoder(truncate, co)
}

func shouldUseColorOutput(
	env Env,
	printOptions options_print.Options,
	stream fd.Std,
) bool {
	if env.GetOptions().IgnoreTtyState {
		return printOptions.PrintColors
	}
	return stream.IsTty() && printOptions.PrintColors
}
