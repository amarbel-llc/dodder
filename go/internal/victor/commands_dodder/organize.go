package commands_dodder

import (
	"os"
	"sync"

	"code.linenisgreat.com/dodder/go/internal/0/orgie_mode"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/env_ui"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_local"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/queries"
	"code.linenisgreat.com/dodder/go/internal/lima/orgie"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/tango/repo_actions"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
	"code.linenisgreat.com/dodder/go/lib/0/vim_cli_options_builder"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/quiter_set"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
	"code.linenisgreat.com/dodder/go/lib/echo/script_value"
)

func init() {
	utility.AddCmd(
		"organize",
		&Organize{
			Flags: orgie.MakeFlags(),
		})
}

func (cmd Organize) GetDescription() command.Description {
	return command.Description{
		Short: "organize objects with a text editor",
		Long: "Open a structured text representation of matching objects " +
			"in your editor. The organize-text format groups objects " +
			"under tag headings. Edits to the text are applied back " +
			"to the store: moving objects between headings changes " +
			"their tags, editing descriptions updates metadata, and " +
			"deleting lines removes tags. See organize-text(7) for " +
			"the format specification.",
	}
}

// Refactor and fold components into userops
type Organize struct {
	command_components_dodder.LocalWorkingCopy
	command_components_dodder.Query

	complete command_components_dodder.Complete

	Flags orgie.Flags
	Mode  orgie_mode.Mode

	Filter script_value.ScriptValue
}

var (
	_ interfaces.CommandComponentWriter = (*Organize)(nil)
	_ command.CommandWithArgs           = (*Organize)(nil)
)

func (cmd *Organize) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{cmd.Query.GetArgGroup()}
}

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

	queryGroup := cmd.MakeQueryResolvingFilenames(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionRequireNonEmptyQuery(),
			queries.BuilderOptionDefaultGenres(genres.Zettel),
			queries.BuilderOptionDefaultSigil(ids.SigilLatest),
		),
		repo,
		req.PopArgs(),
	)

	repo_actions.ApplyToOrganizeOptions(repo, &cmd.Flags.Options)

	objects := sku.MakeSkuTypeSetMutable()
	var lock sync.Mutex

	if err := repo.GetStore().QueryTransactedAsSkuType(
		queryGroup,
		func(checkedOut sku.SkuType) (err error) {
			lock.Lock()
			defer lock.Unlock()

			cloned, _ := checkedOut.Clone() //repool:owned
			return objects.Add(cloned)
		},
	); err != nil {
		repo.Cancel(err)
	}

	defaultQuery := queryGroup.GetDefaultQuery()

	if queryGroup.IsEmpty() && defaultQuery != nil {
		queryGroup = defaultQuery
	}

	createOrganizeFileOp := repo_actions.MakeCreateOrganizeFile(
		repo,
		repo_actions.MakeOrganizeOptionsWithQueryGroup(
			repo,
			cmd.Flags,
			queryGroup,
		),
	)

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
	case orgie_mode.ModeCommitDirectly:
		ui.Log().Print("neither stdin or stdout is a tty")
		ui.Log().Print("generate organize, read from stdin, commit")

		var createOrganizeFileResults *orgie.Text

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

		var organizeText *orgie.Text

		readOrganizeTextOp := repo_actions.MakeReadOrganizeFile(repo)

		{
			var err error

			if organizeText, err = readOrganizeTextOp.Run(
				os.Stdin,
				orgie.NewMetadata(queryGroup.RepoId),
			); err != nil {
				repo.Cancel(err)
			}
		}

		if _, err := repo_actions.LockAndCommitOrganizeResults(
			repo,
			orgie.OrganizeResults{
				Before:     createOrganizeFileResults,
				After:      organizeText,
				Original:   objects,
				QueryGroup: queryGroup,
			},
		); err != nil {
			repo.Cancel(err)
		}

	case orgie_mode.ModeOutputOnly:
		ui.Log().Print("generate organize file and write to stdout")
		if _, err := createOrganizeFileOp.RunAndWrite(os.Stdout); err != nil {
			repo.Cancel(err)
		}

	case orgie_mode.ModeInteractive:
		ui.Log().Print(
			"generate temp file, write organize, open vim to edit, commit results",
		)
		var createOrganizeFileResults *orgie.Text

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

		var organizeText *orgie.Text

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

		if _, err := repo_actions.LockAndCommitOrganizeResults(
			repo,
			orgie.OrganizeResults{
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
	results *orgie.Text,
	queryGroup *queries.Query,
) (ot *orgie.Text, err error) {
	openVimOp := repo_actions.MakeOpenEditor(repo)
	openVimOp.VimOptions = vim_cli_options_builder.New().
		WithFileType("dodder-organize").
		Build()

	if err = openVimOp.Run(path); err != nil {
		err = errors.Wrap(err)
		return ot, err
	}

	readOrganizeTextOp := repo_actions.MakeReadOrganizeFile(repo)

	if ot, err = readOrganizeTextOp.RunWithPath(
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
	var errorRead orgie.ErrorRead

	if err != nil && !errors.As(err, &errorRead) {
		ui.Err().Printf("unrecoverable organize read failure: %s", err)
		tryAgain = false
		return tryAgain
	}

	tryAgain = envUI.Retry("reading changes failed", "edit and retry?", err)

	return tryAgain
}
