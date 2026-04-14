package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/0/checkout_mode"
	"code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/kilo/queries"
	"code.linenisgreat.com/dodder/go/internal/tango/repo_actions"
	"code.linenisgreat.com/dodder/go/internal/uniform/command_components_dodder"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

func init() {
	utility.AddCmd(
		"checkout",
		&Checkout{
			CheckoutOptions: checkout_options.Options{
				CheckoutMode: checkout_mode.Make(checkout_mode.Default),
			},
		})
}

type Checkout struct {
	command_components_dodder.LocalWorkingCopyWithQueryGroup

	CheckoutOptions checkout_options.Options
	Organize        bool
}

var (
	_ interfaces.CommandComponentWriter = (*Checkout)(nil)
	_ command.CommandWithArgs           = (*Checkout)(nil)
)

func (cmd *Checkout) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{cmd.Query.GetArgGroup()}
}

func (cmd Checkout) GetDescription() command.Description {
	return command.Description{
		Short: "check out objects to the workspace",
		Long: "Copy objects from the store into the workspace as editable " +
			"files. Arguments are doddish query terms selecting which " +
			"objects to check out. The workspace must exist (see " +
			"init-workspace). Use -organize to open the checked-out " +
			"objects in an organize-text editor session.",
	}
}

func (cmd *Checkout) SetFlagDefinitions(
	flagDefinitions interfaces.CLIFlagDefinitions,
) {
	cmd.LocalWorkingCopyWithQueryGroup.SetFlagDefinitions(flagDefinitions)
	flagDefinitions.BoolVar(&cmd.Organize, "organize", false, "")
	cmd.CheckoutOptions.SetFlagDefinitions(flagDefinitions)
}

func (cmd Checkout) Run(req command.Request) {
	repo := cmd.MakeLocalWorkingCopy(req)
	envWorkspace := repo.GetEnvWorkspace()

	queryGroup := cmd.MakeQueryIncludingWorkspace(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionPermittedSigil(ids.SigilLatest),
			queries.BuilderOptionPermittedSigil(ids.SigilHidden),
			queries.BuilderOptionRequireNonEmptyQuery(),
			queries.BuilderOptionWorkspace(repo),
			queries.BuilderOptionDefaultGenres(genres.Zettel),
		),
		repo,
		req.PopArgs(),
	)

	opCheckout := repo_actions.MakeCheckout(repo)
	opCheckout.Organize = cmd.Organize
	opCheckout.Options = cmd.CheckoutOptions

	envWorkspace.AssertNotTemporaryOrOfferToCreate(repo)

	if _, err := opCheckout.RunQuery(queryGroup); err != nil {
		repo.Cancel(err)
	}
}
