package commands_dodder

import (
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/delta/genres"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/ids"
	"code.linenisgreat.com/dodder/go/internal/juliett/env_local"
	"code.linenisgreat.com/dodder/go/internal/kilo/command"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku"
	"code.linenisgreat.com/dodder/go/internal/oscar/queries"
	"code.linenisgreat.com/dodder/go/internal/xray/user_ops"
	"code.linenisgreat.com/dodder/go/internal/yankee/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/delta/files"
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
	localWorkingCopy, queryGroup := cmd.MakeLocalWorkingCopyAndQueryGroup(
		dep,
		queries.BuilderOptions(
			queries.BuilderOptionRequireNonEmptyQuery(),
			queries.BuilderOptionDefaultSigil(ids.SigilExternal),
			queries.BuilderOptionDefaultGenres(genres.All()...),
		),
	)

	workspace := localWorkingCopy.GetEnvWorkspace()
	workspaceTags := workspace.GetDefaults().GetDefaultTags()

	for tag := range workspaceTags.All() {
		cmd.Proto.Metadata.AddTagPtr(tag)
	}

	op := user_ops.Checkin{
		Delete:             cmd.Delete,
		Organize:           cmd.Organize,
		Proto:              cmd.Proto,
		CheckoutBlobAndRun: cmd.CheckoutBlobAndRun,
		OpenBlob:           cmd.OpenBlob,
	}

	// TODO add auto dot operator
	if err := op.Run(localWorkingCopy, queryGroup); err != nil {
		dep.Cancel(err)
	}
}
