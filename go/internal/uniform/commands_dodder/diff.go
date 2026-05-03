package commands_dodder

import (
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/charlie/env_local"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/echo/object_metadata_fmt_hyphence"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/sierra/repo_actions"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/delta/files"
)

// TODO switch to registerCommandWithExternalQuery
func init() {
	utility.AddCmd("diff", &Diff{})
}

type Diff struct {
	command_components_dodder.LocalWorkingCopyWithQueryGroup
}

var (
	_ interfaces.CommandComponentWriter = (*Diff)(nil)
	_ command.CommandWithArgs           = (*Diff)(nil)
)

func (cmd *Diff) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{cmd.Query.GetArgGroup()}
}

func (cmd Diff) GetDescription() command.Description {
	return command.Description{
		Short: "show differences between workspace and store",
	}
}

func (cmd *Diff) SetFlagDefinitions(f interfaces.CLIFlagDefinitions) {
	cmd.LocalWorkingCopyWithQueryGroup.SetFlagDefinitions(f)
}

// TODO filter to checked out objects, tags, and types
func (cmd *Diff) Complete(
	_ command.Request,
	envLocal env_local.Env,
	commandLine command.CommandLineInput,
) {
	searchDir := envLocal.GetCwd()

	if commandLine.InProgress != "" && files.Exists(commandLine.InProgress) {
		var err error

		if commandLine.InProgress, err = filepath.Abs(commandLine.InProgress); err != nil {
			envLocal.Cancel(err)
			return
		}

		if searchDir, err = filepath.Rel(searchDir, commandLine.InProgress); err != nil {
			envLocal.Cancel(err)
			return
		}
	}

	for dirEntry, err := range files.WalkDir(searchDir) {
		if err != nil {
			envLocal.Cancel(err)
			return
		}
		if dirEntry.IsDir() {
			continue
		}

		if files.WalkDirIgnoreFuncHidden(dirEntry) {
			continue
		}

		envLocal.GetUI().Printf("%s\tfile", dirEntry.RelPath)
	}
}

func (cmd Diff) Run(dep command.Request) {
	localWorkingCopy, queryGroup := cmd.MakeLocalWorkingCopyAndQueryGroupResolvingFilenames(
		dep,
		queries.BuilderOptions(
			queries.BuilderOptionHidden(nil),
			queries.BuilderOptionDefaultGenres(genres.All()...),
		),
	)

	o := checkout_options.TextFormatterOptions{
		DoNotWriteEmptyDescription: true,
	}

	opDiffFS := repo_actions.MakeDiff(
		localWorkingCopy,
		object_metadata_fmt_hyphence.Factory{
			EnvDir:    localWorkingCopy.GetEnvRepo(),
			BlobStore: localWorkingCopy.GetBlobStore(),
		}.MakeFormatterFamily(),
	)

	if err := localWorkingCopy.GetStore().QuerySkuType(
		queryGroup,
		func(co sku.SkuType) (err error) {
			if err = opDiffFS.Run(co, o); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return err
		},
	); err != nil {
		localWorkingCopy.Cancel(err)
	}
}
