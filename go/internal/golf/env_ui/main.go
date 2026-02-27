package env_ui

import (
	"io"
	"os"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
	"code.linenisgreat.com/dodder/go/internal/charlie/options_print"
	"code.linenisgreat.com/dodder/go/lib/delta/debug"
	"code.linenisgreat.com/dodder/go/internal/delta/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/fd"
)

// TODO explore storing buffered writer and reader
type Env interface {
	// TODO remove and keep separate
	errors.Context

	GetOptions() Options
	GetIn() fd.Std
	GetInFile() io.Reader
	GetUI() fd.Std
	GetUIFile() interfaces.WriterAndStringWriter
	GetOut() fd.Std
	GetOutFile() interfaces.WriterAndStringWriter
	GetErr() fd.Std
	GetErrFile() interfaces.WriterAndStringWriter
	GetCLIConfig() domain_interfaces.CLIConfigProvider

	Confirm(title, description string) (success bool)
	Retry(header, retry string, err error) (tryAgain bool)

	FormatOutputOptions(
		options_print.Options,
	) (o string_format_writer.OutputOptions)
	FormatColorOptionsOut(
		options_print.Options,
	) (o string_format_writer.ColorOptions)
	FormatColorOptionsErr(
		options_print.Options,
	) (o string_format_writer.ColorOptions)
	StringFormatWriterFields(
		truncate string_format_writer.CliFormatTruncation,
		co string_format_writer.ColorOptions,
	) interfaces.StringEncoderTo[string_format_writer.Box]
}

type env struct {
	errors.Context

	options Options

	in  fd.Std
	ui  fd.Std
	out fd.Std
	err fd.Std

	debug *debug.Context

	cliConfig domain_interfaces.CLIConfigProvider
}

func MakeDefault(ctx errors.Context) *env {
	return Make(
		ctx,
		nil,
		debug.Options{},
		Options{},
	)
}

func Make(
	context errors.Context,
	cliConfig domain_interfaces.CLIConfigProvider,
	debugOptions debug.Options,
	options Options,
) *env {
	// TODO use ui printing prefix
	env := &env{
		Context:   context,
		options:   options,
		in:        fd.MakeStd(os.Stdin),
		cliConfig: cliConfig,
	}

	if options.CustomOut != nil {
		env.out = fd.MakeStdFromWriter(options.CustomOut)
	} else {
		env.out = fd.MakeStd(os.Stdout)
	}

	if options.CustomErr != nil {
		env.err = fd.MakeStdFromWriter(options.CustomErr)
	} else {
		env.err = fd.MakeStd(os.Stderr)
	}

	if options.UIFileIsStderr {
		env.ui = env.err
	} else {
		env.ui = env.out
	}

	{
		var err error

		if env.debug, err = debug.MakeContext(context, debugOptions); err != nil {
			context.Cancel(err)
		}
	}

	if cliConfig != nil && cliConfig.GetVerbose() && !cliConfig.GetQuiet() {
		ui.SetVerbose(true)
	} else {
		ui.SetOutput(io.Discard)
	}

	if cliConfig != nil && cliConfig.GetTodo() {
		ui.SetTodoOn()
	}

	return env
}

func (env env) GetOptions() Options {
	return env.options
}

func (env *env) GetIn() fd.Std {
	return env.in
}

func (env *env) GetInFile() io.Reader {
	return env.in.GetFile()
}

func (env *env) GetUI() fd.Std {
	return env.ui
}

func (env *env) GetUIFile() interfaces.WriterAndStringWriter {
	return env.ui.GetFile()
}

func (env *env) GetOut() fd.Std {
	return env.out
}

func (env *env) GetOutFile() interfaces.WriterAndStringWriter {
	return env.out.GetFile()
}

func (env *env) GetErr() fd.Std {
	return env.err
}

func (env *env) GetErrFile() interfaces.WriterAndStringWriter {
	return env.err.GetFile()
}

func (env *env) GetCLIConfig() domain_interfaces.CLIConfigProvider {
	return env.cliConfig
}
