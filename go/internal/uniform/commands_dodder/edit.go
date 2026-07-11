package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/0/checkout_mode"
	"code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/repo_id"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/sierra/repo_actions"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	env_local "github.com/amarbel-llc/madder/go/pkgs/env_local"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

func init() {
	utility.AddCmd(
		"edit",
		&Edit{
			CheckoutMode: checkout_mode.Make(checkout_mode.Default),
		},
	)
}

type Edit struct {
	command_components_dodder.LocalWorkingCopyWithQueryGroup

	complete command_components_dodder.Complete

	// TODO-P3 add force
	command_components_dodder.Checkout
	CheckoutMode checkout_mode.Mode

	// Ephemeral, when set, edits against a temp repo-backed workspace
	// pulled from a resolved parent repo rather than requiring a persistent
	// .dodder-workspace (FDR-0023). ParentPath is the explicit -parent path;
	// empty means the home repo.
	Ephemeral  bool
	ParentPath string

	ephemeral command_components_dodder.ParentBackedWorkspace
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

	flagSet.BoolVar(
		&cmd.Ephemeral,
		"ephemeral",
		false,
		"edit against a temp repo-backed workspace pulled from a resolved parent repo, then push the change back and tear the temp workspace down (no persistent .dodder-workspace required)",
	)

	flagSet.StringVar(
		&cmd.ParentPath,
		"parent",
		"",
		"path to a CWD-scoped parent dodder repository for -ephemeral (omit for home repo)",
	)
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
	if cmd.Ephemeral {
		cmd.runEphemeral(req)
		return
	}

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

// runEphemeral implements FDR-0023: it materializes a temp repo-backed
// workspace whose blob store points at a resolved parent repo, pulls the
// queried object(s) into it, runs the editor against it, pushes the change
// back to the parent, then tears the temp workspace down. On push failure the
// temp workspace is preserved and its path surfaced so no edit is lost.
func (cmd Edit) runEphemeral(req command.Request) {
	// Workspace repos are always CWD-rooted; here the CWD is the temp dir.
	// Mutate the SHARED *repo_config_cli.Config (not a FromAny copy) so
	// Genesis.OnTheFirstDay — which reads config via its own FromAny — sees
	// the CWD-scoped RepoId. A FromAny copy would leave genesis reading the
	// original auto/unknown-location id, which resolves to the XDG home
	// fallback and roots .dodder/.madder (and the pointer blob store) OUTSIDE
	// the temp dir. This mirrors runExperimentalRepo in init_workspace.go.
	config, ok := req.Utility.GetConfigAny().(*repo_config_cli.Config)
	if !ok {
		req.Cancel(
			errors.ErrorWithStackf(
				"expected *repo_config_cli.Config, got %T",
				req.Utility.GetConfigAny(),
			),
		)
		return
	}

	// The user's -repo_id (in config.RepoId) selects the PARENT repo; capture
	// its full spelling (String(), NOT GetName()) before config.RepoId is
	// overwritten with CwdDefault below. GetName() drops the scope dots, so a
	// cwd-scoped id like `.notes` / `..notes` would be re-parsed as a bare
	// XDG-user name and mis-resolved to the home repo (#343 step 5); String()
	// preserves the leading dots that Set() parses back to the cwd scope. Auto
	// (no -repo_id) leaves the parent to -parent / home resolution.
	if !repo_id.IsAuto(config.RepoId) {
		cmd.ephemeral.ParentRepoId = config.RepoId.String()
	}

	config.RepoId = repo_id.CwdDefault()
	cmd.ephemeral.ParentPath = cmd.ParentPath

	queryArgs := req.PopArgs()

	// The shared lifecycle (temp repo-backed workspace, pull, push, teardown)
	// lives in ParentBackedWorkspace.RunEphemeral; the edit-specific step is
	// the checkout/edit against the fresh workspace working copy.
	cmd.ephemeral.RunEphemeral(
		req,
		queryArgs,
		func(edited *local_working_copy.Repo) error {
			editQueryGroup := cmd.ephemeral.Query.MakeQueryIncludingWorkspace(
				req,
				queries.BuilderOptions(
					queries.BuilderOptionDefaultGenres(
						genres.Tag,
						genres.Zettel,
						genres.Type,
						genres.Repo,
					),
				),
				edited,
				queryArgs,
			)

			opEdit := repo_actions.MakeCheckout(edited)
			opEdit.Options = checkout_options.Options{CheckoutMode: cmd.CheckoutMode}
			opEdit.Edit = true
			opEdit.RefreshCheckout = true

			_, err := opEdit.RunQuery(editQueryGroup)
			return err
		},
	)
}
