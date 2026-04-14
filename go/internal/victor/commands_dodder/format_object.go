package commands_dodder

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/0/checkout_mode"
	"code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/object_metadata_fmt_hyphence"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/juliett/typed_blob_store"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/script_config"
)

func init() {
	utility.AddCmd(
		"format-object",
		&FormatObject{
			CheckoutMode: checkout_mode.Make(checkout_mode.Blob),
		})
}

func (cmd FormatObject) GetDescription() command.Description {
	return command.Description{
		Short: "format an object with a type formatter",
	}
}

type FormatObject struct {
	command_components_dodder.LocalWorkingCopy

	CheckoutMode checkout_mode.Mode // add test that says this is unused for stdin
	Stdin        bool               // switch to using `-`
	ids.RepoId
	UTIGroup string
	// TODO add lockfile override option
}

var (
	_ interfaces.CommandComponentWriter = (*FormatObject)(nil)
	_ command.CommandWithArgs           = (*FormatObject)(nil)
)

func (cmd *FormatObject) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{
			{
				Name:        "object-id",
				Description: "object to format",
				Required:    true,
			},
			{
				Name:        "format-id",
				Description: "formatter to use (defaults to type's default)",
			},
		},
	}}
}

func (cmd *FormatObject) SetFlagDefinitions(flagDefs interfaces.CLIFlagDefinitions) {
	flagDefs.BoolVar(
		&cmd.Stdin,
		"stdin",
		false,
		"Read object from stdin and use a Type directly",
	)

	flagDefs.Var(&cmd.RepoId, "kasten", "none or Browser")

	flagDefs.StringVar(&cmd.UTIGroup, "uti-group", "", "lookup format from UTI group")

	flagDefs.Var(&cmd.CheckoutMode, "mode", "mode for checking out the zettel")
}

func (cmd *FormatObject) Run(req command.Request) {
	args := req.PopArgs()
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

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

		if object, err = localWorkingCopy.GetZettelFromObjectId(
			objectIdString,
		); err != nil {
			localWorkingCopy.Cancel(err)
		}
	}

	var typeObject *sku.Transacted

	{
		var err error

		if typeObject, err = localWorkingCopy.GetStore().ReadObjectTypeAndLockIfNecessary(
			object,
		); err != nil {
			localWorkingCopy.Cancel(err)
		}
	}

	{
		var err error

		if blobFormatter, err = localWorkingCopy.GetBlobFormatter(
			typeObject,
			formatId,
			cmd.UTIGroup,
		); err != nil {
			localWorkingCopy.Cancel(err)
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

	formatter := typed_blob_store.MakeTextFormatterWithBlobFormatter(
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

	if _, err := formatter.WriteStringFormatWithMode(
		localWorkingCopy.GetUIFile(),
		object,
		cmd.CheckoutMode,
	); err != nil {
		var errBlobFormatterFailed *object_metadata_fmt_hyphence.ErrBlobFormatterFailed

		if errors.As(err, &errBlobFormatterFailed) {
			localWorkingCopy.Cancel(errBlobFormatterFailed)
			// err = nil
			// ui.Err().Print(errExit)
		} else {
			localWorkingCopy.Cancel(err)
		}
	}
}

func (cmd *FormatObject) FormatFromStdin(
	repo *local_working_copy.Repo,
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
		if typeObject, err = repo.GetStore().ReadOneObjectId(typeLock.GetKey()); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else {
		if typeObject, err = repo.GetStore().ReadTypeObject(&typeLock); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	if blobFormatter, err = repo.GetBlobFormatter(
		typeObject,
		formatId,
		cmd.UTIGroup,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var blobTreeDir string

	if blobTreeDir, err = repo.MaterializeBlobTree(
		typeObject,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	env := repo.GetEnvRepo().MakeCommonEnv()

	if blobTreeDir != "" {
		env["DODDER_BLOB_TREE"] = blobTreeDir
	}

	var wt io.WriterTo

	if wt, err = script_config.MakeWriterToWithStdin(
		blobFormatter,
		env,
		repo.GetInFile(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if _, err = wt.WriteTo(repo.GetUIFile()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
