package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/bravo/checkout_mode"
	"code.linenisgreat.com/dodder/go/internal/charlie/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
	"code.linenisgreat.com/dodder/go/internal/november/queries"
	"code.linenisgreat.com/dodder/go/internal/whiskey/user_ops"
	"code.linenisgreat.com/dodder/go/internal/xray/command_components_dodder"
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

var _ interfaces.CommandComponentWriter = (*Checkout)(nil)

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

	opCheckout := user_ops.Checkout{
		Repo:     repo,
		Organize: cmd.Organize,
		Options:  cmd.CheckoutOptions,
	}

	envWorkspace.AssertNotTemporaryOrOfferToCreate(repo)

	if _, err := opCheckout.RunQuery(queryGroup); err != nil {
		repo.Cancel(err)
	}
}
