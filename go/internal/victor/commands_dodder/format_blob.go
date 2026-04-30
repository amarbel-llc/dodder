package commands_dodder

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/0/checkout_mode"
	"code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_local"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/juliett/typed_blob_store"
	"code.linenisgreat.com/dodder/go/internal/kilo/queries"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/script_config"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

func init() {
	utility.AddCmd("format-blob", &FormatBlob{})
}

func (cmd FormatBlob) GetDescription() command.Description {
	return command.Description{
		Short: "format an object's blob content",
	}
}

type FormatBlob struct {
	command_components_dodder.LocalWorkingCopy

	complete command_components_dodder.Complete

	Stdin    bool
	UTIGroup string
}

var (
	_ interfaces.CommandComponentWriter = (*FormatBlob)(nil)
	_ command.CommandWithArgs           = (*FormatBlob)(nil)
)

func (cmd *FormatBlob) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{
			{
				Name:        "object-id",
				Description: "object whose blob to format",
				Required:    true,
			},
			{
				Name:        "format-id",
				Description: "formatter to use (defaults to type's default)",
			},
		},
	}}
}

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

	var typeObject *sku.Transacted

	{
		var err error

		if typeObject, err = localWorkingCopy.GetStore().ReadObjectTypeAndLockIfNecessary(
			object,
		); err != nil {
			errors.ContextCancelWithErrorAndFormat(
				localWorkingCopy,
				err,
				"objectIdString: %q, Object: %q",
				objectIdString, sku.String(object),
			)
		}
	}

	{
		var err error

		if blobFormatter, err = localWorkingCopy.GetBlobFormatter(
			typeObject,
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

	var blobTreeDir string

	{
		var err error

		if blobTreeDir, err = localWorkingCopy.MaterializeBlobTree(
			typeObject,
		); err != nil {
			localWorkingCopy.Cancel(err)
		}
	}

	format := typed_blob_store.MakeTextFormatterWithBlobFormatter(
		localWorkingCopy.GetEnvRepo(),
		checkout_options.TextFormatterOptions{
			DoNotWriteEmptyDescription: true,
		},
		localWorkingCopy.GetConfig(),
		blobFormatter,
		blobTreeDir,
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

	var typeObject *sku.Transacted

	if typeLock.GetValue().IsNull() {
		if typeObject, err = u.GetStore().ReadOneObjectId(typeLock.GetKey()); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else {
		if typeObject, err = u.GetStore().ReadTypeObject(&typeLock); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if blobFormatter, err = u.GetBlobFormatter(
		typeObject,
		formatId,
		cmd.UTIGroup,
	); err != nil {
		u.Cancel(err)
	}

	var blobTreeDir string

	if blobTreeDir, err = u.MaterializeBlobTree(
		typeObject,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	env := u.GetEnvRepo().MakeCommonEnv()

	if blobTreeDir != "" {
		env["DODDER_BLOB_TREE"] = blobTreeDir
	}

	var wt io.WriterTo

	if wt, err = script_config.MakeWriterToWithStdin(
		blobFormatter,
		env,
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
