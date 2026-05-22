package commands_dodder

import (
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/0/haustoria"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	env_local "github.com/amarbel-llc/madder/go/pkgs/env_local"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/sierra/repo_actions"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/files"
)

func init() {
	cmd := &Checkin{
		Proto: sku.MakeProto(nil),
	}

	utility.AddCmd("checkin", cmd)
	utility.AddCmd("add", cmd)
	utility.AddCmd("save", cmd)
}

type Checkin struct {
	command_components_dodder.LocalWorkingCopyWithQueryGroup

	complete command_components_dodder.Complete

	IgnoreBlob bool
	Proto      sku.Proto

	command_components_dodder.Checkout

	CheckoutBlobAndRun string
	OpenBlob           bool
}

var _ interfaces.CommandComponentWriter = (*Checkin)(nil)

func (cmd Checkin) GetDescription() command.Description {
	return command.Description{
		Short: "commit workspace changes to the store",
		Long: "Commit checked-out objects from the workspace back into the " +
			"store. With no arguments, commits all modified objects. " +
			"Query arguments filter which objects to commit. Use " +
			"-description, -tags, and -type to set metadata on new " +
			"zettels created during checkin. Use -each-blob to run an " +
			"external command on each blob before committing. Also " +
			"available as 'add' and 'save'.",
	}
}

func (cmd *Checkin) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{cmd.Query.GetArgGroup()}
}

func (cmd *Checkin) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	cmd.LocalWorkingCopyWithQueryGroup.SetFlagDefinitions(flagSet)

	flagSet.BoolVar(
		&cmd.IgnoreBlob,
		"ignore-blob",
		false,
		"do not change the blob",
	)

	flagSet.StringVar(
		&cmd.CheckoutBlobAndRun,
		"each-blob",
		"",
		"checkout each Blob and run a utility",
	)

	cmd.complete.SetFlagsProto(
		&cmd.Proto,
		flagSet,
		"description to use for new zettels",
		"tags added for new zettels",
		"type used for new zettels",
	)

	cmd.Checkout.SetFlagDefinitions(flagSet)
}

// TODO refactor into common
func (cmd *Checkin) Complete(
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

		if files.WalkDirIgnoreFuncHidden(dirEntry) {
			continue
		}

		if !dirEntry.IsDir() {
			envLocal.GetUI().Printf("%s\tfile", dirEntry.RelPath)
		} else {
			envLocal.GetUI().Printf("%s/\tdirectory", dirEntry.RelPath)
		}
	}
}

func (cmd Checkin) Run(dep command.Request) {
	localWorkingCopy, queryGroup := cmd.MakeLocalWorkingCopyAndQueryGroupResolvingFilenames(
		dep,
		queries.BuilderOptions(
			queries.BuilderOptionRequireNonEmptyQuery(),
			queries.BuilderOptionDefaultSigil(ids.SigilExternal),
			queries.BuilderOptionDefaultGenres(genres.All()...),
		),
	)

	workspace := localWorkingCopy.GetEnvWorkspace()

	if h, ok := workspace.GetStore().StoreLike.(haustoria.Haustoria); ok {
		op := repo_actions.MakeCheckinHaustoria(
			localWorkingCopy,
			h,
			workspace.GetStore().StoreLike,
			queryGroup,
		)

		if _, err := op.Run(); err != nil {
			dep.Cancel(err)
		}

		return
	}

	workspaceTags := workspace.GetDefaults().GetDefaultTags()

	for tag := range workspaceTags.All() {
		cmd.Proto.Metadata.AddTagPtr(tag)
	}

	op := repo_actions.MakeCheckin(localWorkingCopy)
	op.Delete = cmd.Delete
	op.Organize = cmd.Organize
	op.Proto = cmd.Proto
	op.CheckoutBlobAndRun = cmd.CheckoutBlobAndRun
	op.OpenBlob = cmd.OpenBlob

	// TODO add auto dot operator
	if err := op.Run(queryGroup); err != nil {
		dep.Cancel(err)
	}
}
