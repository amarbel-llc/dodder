package commands_dodder

import (
	"io"
	"os"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/orgie"
	"code.linenisgreat.com/dodder/go/internal/sierra/repo_actions"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"github.com/amarbel-llc/madder/go/pkgs/fd"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/files"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

func init() {
	utility.AddCmd(
		"format-organize",
		&FormatOrganize{
			Flags: orgie.MakeFlags(),
		},
	)
}

func (cmd FormatOrganize) GetDescription() command.Description {
	return command.Description{
		Short: "format an organize file",
	}
}

type FormatOrganize struct {
	command_components_dodder.LocalWorkingCopy

	Flags orgie.Flags
}

var (
	_ interfaces.CommandComponentWriter = (*FormatOrganize)(nil)
	_ command.CommandWithArgs           = (*FormatOrganize)(nil)
)

func (cmd *FormatOrganize) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{{
			Name:        "file-descriptor",
			Description: "file descriptor of the organize file",
			Required:    true,
		}},
	}}
}

func (cmd *FormatOrganize) SetFlagDefinitions(f interfaces.CLIFlagDefinitions) {
	cmd.Flags.SetFlagDefinitions(f)
}

func (cmd *FormatOrganize) Run(dep command.Request) {
	args := dep.PopArgs()
	localWorkingCopy := cmd.MakeLocalWorkingCopy(dep)

	cmd.Flags.Config = localWorkingCopy.GetConfigPtr()

	if len(args) != 1 {
		errors.ContextCancelWithErrorf(
			localWorkingCopy,
			"expected exactly one input argument",
		)
	}

	var fdee fd.FD

	if err := fdee.Set(args[0]); err != nil {
		localWorkingCopy.Cancel(err)
	}

	var r io.Reader

	if fdee.IsStdin() {
		r = os.Stdin
	} else {
		var f *os.File

		{
			var err error

			if f, err = files.Open(args[0]); err != nil {
				localWorkingCopy.Cancel(err)
			}
		}

		r = f

		defer errors.ContextMustClose(localWorkingCopy, f)
	}

	var ot *orgie.Text

	readOrganizeTextOp := repo_actions.MakeReadOrganizeFile(localWorkingCopy)

	var repoId ids.RepoId

	{
		var err error

		if ot, err = readOrganizeTextOp.Run(
			r,
			orgie.NewMetadata(repoId),
		); err != nil {
			localWorkingCopy.Cancel(err)
		}
	}

	ot.Options = cmd.Flags.GetOptionsWithMetadata(
		localWorkingCopy.GetConfig().GetPrintOptions(),
		localWorkingCopy.SkuFormatBoxCheckedOutNoColor(),
		localWorkingCopy.GetStore().GetAbbrStore().GetAbbr(),
		sku.ObjectFactory{},
		ot.Metadata,
	)

	if err := ot.Refine(); err != nil {
		localWorkingCopy.Cancel(err)
	}

	if _, err := ot.WriteTo(os.Stdout); err != nil {
		localWorkingCopy.Cancel(err)
	}
}
