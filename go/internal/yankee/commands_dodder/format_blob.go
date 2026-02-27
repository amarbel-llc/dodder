package commands_dodder

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/checkout_mode"
	"code.linenisgreat.com/dodder/go/internal/charlie/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
	"code.linenisgreat.com/dodder/go/internal/charlie/script_config"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/echo/markl"
	"code.linenisgreat.com/dodder/go/internal/india/env_local"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/mike/typed_blob_store"
	"code.linenisgreat.com/dodder/go/internal/november/queries"
	"code.linenisgreat.com/dodder/go/internal/victor/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/xray/command_components_dodder"
)

func init() {
	utility.AddCmd("format-blob", &FormatBlob{})
}

type FormatBlob struct {
	command_components_dodder.LocalWorkingCopy

	complete command_components_dodder.Complete

	Stdin    bool
	UTIGroup string
}

var _ interfaces.CommandComponentWriter = (*FormatBlob)(nil)

func (cmd *FormatBlob) SetFlagDefinitions(f interfaces.CLIFlagDefinitions) {
	f.BoolVar(
		&cmd.Stdin,
		"stdin",
		false,
		"Read object from stdin and use a Type directly",
	)

	f.StringVar(
		&cmd.UTIGroup,
		"uti-group",
		"",
		"lookup format from UTI group",
	)
}

func (cmd *FormatBlob) Complete(
	req command.Request,
	envLocal env_local.Env,
	commandLine command.CommandLineInput,
) {
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	args := commandLine.FlagsOrArgs[1:]

	if commandLine.InProgress != "" {
		args = args[:len(args)-1]
	}

	cmd.complete.CompleteObjects(
		req,
		localWorkingCopy,
		queries.BuilderOptionDefaultGenres(genres.Zettel),
		args...,
	)
}

func (cmd *FormatBlob) Run(dep command.Request) {
	args := dep.PopArgs()
	localWorkingCopy := cmd.MakeLocalWorkingCopy(dep)

	if cmd.Stdin {
		if err := cmd.FormatFromStdin(localWorkingCopy, args...); err != nil {
			localWorkingCopy.Cancel(err)
		}

		return
	}

	var formatId string

	var objectIdString string
	var blobFormatter script_config.RemoteScript

	switch len(args) {
	case 2:
		formatId = args[1]
		fallthrough

	case 1:
		objectIdString = args[0]

	default:
		errors.ContextCancelWithErrorf(
			localWorkingCopy,
			"expected one or two input arguments, but got %d",
			len(args),
		)
	}

	var object *sku.Transacted

	{
		var err error

		if object, err = localWorkingCopy.GetZettelFromObjectId(objectIdString); err != nil {
			localWorkingCopy.Cancel(err)
		}
	}

	typeLock := object.GetMetadata().GetTypeLock()

	{
		var err error

		if blobFormatter, err = localWorkingCopy.GetBlobFormatter(
			typeLock,
			formatId,
			cmd.UTIGroup,
		); err != nil {
			errors.ContextCancelWithErrorAndFormat(
				localWorkingCopy,
				err,
				"objectIdString: %q, Object: %q",
				objectIdString, sku.String(object),
			)
		}
	}

	format := typed_blob_store.MakeTextFormatterWithBlobFormatter(
		localWorkingCopy.GetEnvRepo(),
		checkout_options.TextFormatterOptions{
			DoNotWriteEmptyDescription: true,
		},
		localWorkingCopy.GetConfig(),
		blobFormatter,
		checkout_mode.Make(),
	)

	if err := localWorkingCopy.GetStore().TryFormatHook(object); err != nil {
		localWorkingCopy.Cancel(err)
	}

	if _, err := format.WriteStringFormatWithMode(
		localWorkingCopy.GetUIFile(),
		object,
		checkout_mode.Make(checkout_mode.Blob),
	); err != nil {
		localWorkingCopy.Cancel(err)
	}
}

func (cmd *FormatBlob) FormatFromStdin(
	u *local_working_copy.Repo,
	args ...string,
) (err error) {
	formatId := "text"

	var blobFormatter script_config.RemoteScript
	typeLock := markl.MakeLock[ids.SeqId]()
	typeLockMarshaler := markl.MakeMutableLockCoderValueNotRequired(&typeLock)

	switch len(args) {
	case 1:
		if err = typeLockMarshaler.Set(args[0]); err != nil {
			err = errors.Wrap(err)
			return err
		}

	case 2:
		formatId = args[0]
		if err = typeLockMarshaler.Set(args[1]); err != nil {
			err = errors.Wrap(err)
			return err
		}

	default:
		err = errors.ErrorWithStackf(
			"expected one or two input arguments, but got %d",
			len(args),
		)
		return err
	}

	if blobFormatter, err = u.GetBlobFormatter(
		&typeLock,
		formatId,
		cmd.UTIGroup,
	); err != nil {
		u.Cancel(err)
	}

	var wt io.WriterTo

	if wt, err = script_config.MakeWriterToWithStdin(
		blobFormatter,
		u.GetEnvRepo().MakeCommonEnv(),
		u.GetInFile(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if _, err = wt.WriteTo(u.GetUIFile()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
