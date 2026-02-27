package commands_dodder

import (
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/charlie/files"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
	"code.linenisgreat.com/dodder/go/internal/india/env_local"
	"code.linenisgreat.com/dodder/go/internal/india/object_metadata_fmt_triple_hyphen"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/november/queries"
	"code.linenisgreat.com/dodder/go/internal/whiskey/user_ops"
	"code.linenisgreat.com/dodder/go/internal/xray/command_components_dodder"
)

// TODO switch to registerCommandWithExternalQuery
func init() {
	utility.AddCmd("diff", &Diff{})
}

type Diff struct {
	command_components_dodder.LocalWorkingCopyWithQueryGroup
}

var _ interfaces.CommandComponentWriter = (*Diff)(nil)

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
	localWorkingCopy, queryGroup := cmd.MakeLocalWorkingCopyAndQueryGroup(
		dep,
		queries.BuilderOptions(
			queries.BuilderOptionHidden(nil),
			queries.BuilderOptionDefaultGenres(genres.All()...),
		),
	)

	o := checkout_options.TextFormatterOptions{
		DoNotWriteEmptyDescription: true,
	}

	opDiffFS := user_ops.Diff{
		Repo: localWorkingCopy,
		FormatterFamily: object_metadata_fmt_triple_hyphen.Factory{
			EnvDir:    localWorkingCopy.GetEnvRepo(),
			BlobStore: localWorkingCopy.GetBlobStore(),
		}.MakeFormatterFamily(),
	}

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
