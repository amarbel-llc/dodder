package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/0/checkout_mode"
	"code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	env_local "github.com/amarbel-llc/madder/go/pkgs/env_local"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/sierra/repo_actions"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

func init() {
	utility.AddCmd(
		"edit",
		&Edit{
			CheckoutMode: checkout_mode.Make(checkout_mode.Default),
		})
}

type Edit struct {
	command_components_dodder.LocalWorkingCopyWithQueryGroup

	complete command_components_dodder.Complete

	// TODO-P3 add force
	command_components_dodder.Checkout
	CheckoutMode checkout_mode.Mode
}

var (
	_ interfaces.CommandComponentWriter = (*Edit)(nil)
	_ command.CommandWithArgs           = (*Edit)(nil)
)

func (cmd *Edit) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{cmd.Query.GetArgGroup()}
}

func (cmd Edit) GetDescription() command.Description {
	return command.Description{
		Short: "check out and edit objects in an editor",
		Long: "Check out matching objects, open them in your configured " +
			"editor, and commit changes when the editor exits. This is " +
			"a shortcut combining checkout, edit, and checkin into a " +
			"single operation. Arguments are doddish query terms. Use " +
			"-mode to control the checkout format.",
	}
}

func (cmd *Edit) SetFlagDefinitions(flagSet interfaces.CLIFlagDefinitions) {
	cmd.LocalWorkingCopyWithQueryGroup.SetFlagDefinitions(flagSet)

	cmd.Checkout.SetFlagDefinitions(flagSet)

	flagSet.Var(&cmd.CheckoutMode, "mode", "mode for checking out the object")
}

func (cmd Edit) CompletionGenres() ids.Genre {
	return ids.MakeGenre(
		genres.Tag,
		genres.Zettel,
		genres.Type,
		genres.Repo,
	)
}

func (cmd *Edit) Complete(
	req command.Request,
	envLocal env_local.Env,
	commandLine command.CommandLineInput,
) {
	localWorkingCopy := cmd.MakeLocalWorkingCopy(req)

	args := commandLine.FlagsOrArgs[1:]

	if commandLine.InProgress != "" {
		args = args[:len(args)-1]
	}

	cmd.complete.CompleteObjectsIncludingWorkspace(
		req,
		localWorkingCopy,
		queries.BuilderOptionDefaultGenres(genres.Zettel),
		args...,
	)
}

func (cmd Edit) Run(req command.Request) {
	repo := cmd.MakeLocalWorkingCopy(req)
	envWorkspace := repo.GetEnvWorkspace()

	// TODO eventually remove this once temporary edits work correctly
	envWorkspace.AssertNotTemporaryOrOfferToCreate(repo)

	queryGroup := cmd.MakeQueryIncludingWorkspace(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionWorkspace(repo),
			queries.BuilderOptionDefaultGenres(
				genres.Tag,
				genres.Zettel,
				genres.Type,
				genres.Repo,
			),
		),
		repo,
		req.PopArgs(),
	)

	options := checkout_options.Options{
		CheckoutMode: cmd.CheckoutMode,
	}

	opEdit := repo_actions.MakeCheckout(repo)
	opEdit.Options = options
	opEdit.Edit = true
	opEdit.RefreshCheckout = true

	if _, err := opEdit.RunQuery(queryGroup); err != nil {
		repo.Cancel(err)
	}
}
