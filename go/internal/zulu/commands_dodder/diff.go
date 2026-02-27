package commands_dodder

import (
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/delta/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/delta/genres"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_local"
	"code.linenisgreat.com/dodder/go/internal/juliett/object_metadata_fmt_triple_hyphen"
	"code.linenisgreat.com/dodder/go/internal/kilo/command"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/internal/oscar/queries"
	"code.linenisgreat.com/dodder/go/internal/xray/user_ops"
	"code.linenisgreat.com/dodder/go/internal/yankee/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
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
