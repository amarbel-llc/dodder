package commands_dodder

import (
	"os"
	"sync"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/_/vim_cli_options_builder"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/organize_text_mode"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter_set"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/delta/script_value"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/golf/env_ui"
	"code.linenisgreat.com/dodder/go/src/india/env_local"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/november/queries"
	"code.linenisgreat.com/dodder/go/src/oscar/organize_text"
	"code.linenisgreat.com/dodder/go/src/victor/local_working_copy"
	"code.linenisgreat.com/dodder/go/src/whiskey/user_ops"
	"code.linenisgreat.com/dodder/go/src/xray/command_components_dodder"
)

func init() {
	utility.AddCmd(
		"organize",
		&Organize{
			Flags: organize_text.MakeFlags(),
		})
}

// Refactor and fold components into userops
type Organize struct {
	command_components_dodder.LocalWorkingCopy
	command_components_dodder.Query

	complete command_components_dodder.Complete

	Flags organize_text.Flags
	Mode  organize_text_mode.Mode

	Filter script_value.ScriptValue
}

var _ interfaces.CommandComponentWriter = (*Organize)(nil)

func (cmd *Organize) SetFlagDefinitions(flagDef interfaces.CLIFlagDefinitions) {
	cmd.Query.SetFlagDefinitions(flagDef)

	cmd.Flags.SetFlagDefinitions(flagDef)

	flagDef.Var(
		&cmd.Filter,
		"filter",
		"a script to run for each file to transform it the standard zettel format",
	)

	flagDef.Var(&cmd.Mode, "mode", "mode used for handling stdin and stdout")
}

func (cmd *Organize) CompletionGenres() ids.Genre {
	return ids.MakeGenre(
		genres.Zettel,
		genres.Tag,
		genres.Type,
	)
}

func (cmd Organize) Complete(
	req command.Request,
	envLocal env_local.Env,
	commandLine command.CommandLineInput,
) {
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	args := commandLine.FlagsOrArgs.Shift(1)

	if commandLine.InProgress != "" {
		args = args[:len(args)-1]
	}

	cmd.complete.CompleteObjects(
		req,
		localWorkingCopy,
		queries.BuilderOptionDefaultGenres(
			genres.Tag,
			genres.Type,
		),
		args...,
	)
}

func (cmd *Organize) Run(req command.Request) {
	repo := cmd.MakeLocalWorkingCopy(req)

	queryGroup := cmd.MakeQueryIncludingWorkspace(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionRequireNonEmptyQuery(),
			queries.BuilderOptionWorkspace(repo),
			queries.BuilderOptionDefaultGenres(genres.Zettel),
			queries.BuilderOptionDefaultSigil(ids.SigilLatest),
		),
		repo,
		req.PopArgs(),
	)

	repo.ApplyToOrganizeOptions(&cmd.Flags.Options)

	objects := sku.MakeSkuTypeSetMutable()
	var lock sync.Mutex

	if err := repo.GetStore().QueryTransactedAsSkuType(
		queryGroup,
		func(checkedOut sku.SkuType) (err error) {
			lock.Lock()
			defer lock.Unlock()

			cloned, _ := checkedOut.Clone()
			return objects.Add(cloned)
		},
	); err != nil {
		repo.Cancel(err)
	}

	defaultQuery := queryGroup.GetDefaultQuery()

	if queryGroup.IsEmpty() && defaultQuery != nil {
		queryGroup = defaultQuery
	}

	createOrganizeFileOp := user_ops.CreateOrganizeFile{
		Repo: repo,
		Options: repo.MakeOrganizeOptionsWithQueryGroup(
			cmd.Flags,
			queryGroup,
		),
	}

	createOrganizeFileOp.Skus = objects

	types := queries.GetTypes(queryGroup)

	if types.Len() == 1 {
		createOrganizeFileOp.Type = quiter_set.Any(types)
	}

	tags := queries.GetTags(queryGroup)

	if objects.Len() == 0 {
		workspace := repo.GetEnvWorkspace()
		workspaceTags := workspace.GetDefaults().GetDefaultTags()

		for tag := range workspaceTags.All() {
			ids.TagSetMutableAdd(tags, tag)
		}
	}

	createOrganizeFileOp.TagSet = tags

	switch cmd.Mode {
	case organize_text_mode.ModeCommitDirectly:
		ui.Log().Print("neither stdin or stdout is a tty")
		ui.Log().Print("generate organize, read from stdin, commit")

		var createOrganizeFileResults *organize_text.Text

		var file *os.File

		{
			var err error

			if file, err = repo.GetEnvRepo().GetTempLocal().FileTempWithTemplate(
				"*." + repo.GetConfig().GetFileExtensions().Organize,
			); err != nil {
				repo.Cancel(err)
			}
		}

		defer errors.ContextMustClose(repo, file)

		{
			var err error

			if createOrganizeFileResults, err = createOrganizeFileOp.RunAndWrite(
				file,
			); err != nil {
				repo.Cancel(err)
			}
		}

		var organizeText *organize_text.Text

		readOrganizeTextOp := user_ops.ReadOrganizeFile{}

		{
			var err error

			if organizeText, err = readOrganizeTextOp.Run(
				repo,
				os.Stdin,
				organize_text.NewMetadata(queryGroup.RepoId),
			); err != nil {
				repo.Cancel(err)
			}
		}

		if _, err := repo.LockAndCommitOrganizeResults(
			organize_text.OrganizeResults{
				Before:     createOrganizeFileResults,
				After:      organizeText,
				Original:   objects,
				QueryGroup: queryGroup,
			},
		); err != nil {
			repo.Cancel(err)
		}

	case organize_text_mode.ModeOutputOnly:
		ui.Log().Print("generate organize file and write to stdout")
		if _, err := createOrganizeFileOp.RunAndWrite(os.Stdout); err != nil {
			repo.Cancel(err)
		}

	case organize_text_mode.ModeInteractive:
		ui.Log().Print(
			"generate temp file, write organize, open vim to edit, commit results",
		)
		var createOrganizeFileResults *organize_text.Text

		var f *os.File

		{
			var err error

			if f, err = repo.GetEnvRepo().GetTempLocal().FileTempWithTemplate(
				"*." + repo.GetConfig().GetFileExtensions().Organize,
			); err != nil {
				repo.Cancel(err)
			}

			defer errors.ContextMustClose(repo, f)
		}

		{
			var err error

			if createOrganizeFileResults, err = createOrganizeFileOp.RunAndWrite(
				f,
			); err != nil {
				errors.ContextCancelWithErrorAndFormat(
					repo,
					err,
					"Organize File: %q",
					f.Name(),
				)
			}
		}

		var organizeText *organize_text.Text

		{
			var err error

			if organizeText, err = cmd.readFromVim(
				repo,
				f.Name(),
				createOrganizeFileResults,
				queryGroup,
			); err != nil {
				errors.ContextCancelWithErrorAndFormat(
					repo,
					err,
					"Organize File: %q",
					f.Name(),
				)
			}
		}

		if _, err := repo.LockAndCommitOrganizeResults(
			organize_text.OrganizeResults{
				Before:     createOrganizeFileResults,
				After:      organizeText,
				Original:   objects,
				QueryGroup: queryGroup,
			},
		); err != nil {
			repo.Cancel(err)
		}

	default:
		errors.ContextCancelWithErrorf(repo, "unknown mode")
	}
}

func (cmd Organize) readFromVim(
	repo *local_working_copy.Repo,
	path string,
	results *organize_text.Text,
	queryGroup *queries.Query,
) (ot *organize_text.Text, err error) {
	openVimOp := user_ops.OpenEditor{
		VimOptions: vim_cli_options_builder.New().
			WithFileType("dodder-organize").
			Build(),
	}

	if err = openVimOp.Run(repo, path); err != nil {
		err = errors.Wrap(err)
		return ot, err
	}

	readOrganizeTextOp := user_ops.ReadOrganizeFile{}

	if ot, err = readOrganizeTextOp.RunWithPath(
		repo,
		path,
		queryGroup.RepoId,
	); err != nil {
		if cmd.handleReadChangesError(repo, err) {
			err = nil
			ot, err = cmd.readFromVim(repo, path, results, queryGroup)
		} else {
			ui.Err().Printf("aborting organize")
			return ot, err
		}
	}

	return ot, err
}

// TODO migrate to using errors.Retryable
func (cmd Organize) handleReadChangesError(
	envUI env_ui.Env,
	err error,
) (tryAgain bool) {
	var errorRead organize_text.ErrorRead

	if err != nil && !errors.As(err, &errorRead) {
		ui.Err().Printf("unrecoverable organize read failure: %s", err)
		tryAgain = false
		return tryAgain
	}

	tryAgain = envUI.Retry("reading changes failed", "edit and retry?", err)

	return tryAgain
}
