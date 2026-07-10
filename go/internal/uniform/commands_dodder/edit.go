package commands_dodder

import (
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/0/checkout_mode"
	"code.linenisgreat.com/dodder/go/internal/0/dodder_env"
	"code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/repo_id"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/echo/workspace_config_blobs"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/sierra/repo_actions"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	env_local "github.com/amarbel-llc/madder/go/pkgs/env_local"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/xdg"
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

	config.RepoId = repo_id.CwdDefault()

	cmd.ephemeral.ParentPath = cmd.ParentPath
	cmd.ephemeral.Genesis.BigBang.SetDefaults()

	// Workspace repos have no default type (matches init-workspace
	// -experimental-repo); creating new objects there requires an explicit
	// type, but editing existing ones does not.
	cmd.ephemeral.Genesis.BigBang.ExcludeDefaultType = true

	absParentPath, parentIsHomeRepo := cmd.ephemeral.ResolveParentPath(req)
	cmd.ephemeral.ValidateParentRepo(req, absParentPath, parentIsHomeRepo)

	cmd.ephemeral.LinkParentZettelIdProviders(absParentPath, parentIsHomeRepo)

	queryArgs := req.PopArgs()

	originalCwd, err := os.Getwd()
	if err != nil {
		req.Cancel(err)
		return
	}

	tempDir, err := os.MkdirTemp("", "dodder-ephemeral-")
	if err != nil {
		req.Cancel(err)
		return
	}

	// Resolve symlinks so tempDir matches os.Getwd()'s canonical form after
	// chdir (macOS $TMPDIR is a symlink); the ceiling comparison below is
	// path-equality based, so the two must agree.
	if resolved, resolveErr := filepath.EvalSymlinks(tempDir); resolveErr == nil {
		tempDir = resolved
	}

	if err = os.Chdir(tempDir); err != nil {
		req.Cancel(err)
		return
	}

	// Pin both ceilings to the temp dir for the rest of this process. The temp
	// dir sits under $TMPDIR, typically OUTSIDE the caller's ceiling; the fresh
	// MakeLocalWorkingCopy below discovers the workspace by walking UP from cwd
	// for .dodder-workspace, and that walk-up honors the ceiling. Without
	// re-pinning to the temp dir, the caller's ceiling (which may sit below
	// tempDir under a sandbox) would cut the walk short and the workspace would
	// not be found.
	if err = os.Setenv(
		xdg.CeilingEnvVarName(dodder_env.XDGUtilityName),
		tempDir,
	); err != nil {
		req.Cancel(err)
		return
	}

	if err = os.Setenv(
		xdg.CeilingEnvVarName(dodder_env.XDGUtilityNameMadder),
		tempDir,
	); err != nil {
		req.Cancel(err)
		return
	}

	// FDR-0005 / FDR-0023: wire a TomlPointerV1 blob store pointing at the
	// parent's default store so the workspace repo holds NO blob copy of its
	// own. Because config.RepoId is CWD-scoped (set above via the shared
	// *Config pointer), genesis roots .dodder/.madder AND this pointer store
	// inside tempDir, so a fresh in-process working copy discovers it.
	cmd.ephemeral.SetupParentPointerBlobStore(
		req,
		"ephemeral",
		absParentPath,
		parentIsHomeRepo,
	)

	local, remote := cmd.ephemeral.CreateRepoAndPullFromParent(
		req,
		absParentPath,
		parentIsHomeRepo,
		queryArgs,
		repo.ImporterOptions{}.WithPrintCopies(true),
	)

	if err = local.GetEnvWorkspace().CreateWorkspace(
		&workspace_config_blobs.V0{},
	); err != nil {
		req.Cancel(err)
		return
	}

	// Flush the genesis repo and open a fresh working copy of the now-written
	// temp workspace: the genesis env was built before .dodder-workspace
	// existed, so it still resolves as a read-only temporary workspace. The
	// fresh open resolves it as writable and re-discovers the pointer blob
	// store from tempDir.
	if err = local.Flush(); err != nil {
		req.Cancel(err)
		return
	}

	edited := cmd.ephemeral.MakeLocalWorkingCopy(req)

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

	if _, err = opEdit.RunQuery(editQueryGroup); err != nil {
		// Preserve the temp workspace so the edit is not lost (FDR-0023).
		req.Cancel(
			errors.Wrapf(
				err,
				"ephemeral edit failed; workspace kept at %s",
				tempDir,
			),
		)
		return
	}

	// Push back to the parent: the remote pulls the whole workspace from the
	// ephemeral local (push is "remote pulls from local"; mirrors push.go).
	pushQueryGroup := cmd.ephemeral.Query.MakeQueryIncludingWorkspace(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionDefaultSigil(
				ids.SigilHistory,
				ids.SigilHidden,
			),
			queries.BuilderOptionDefaultGenres(genres.InventoryList),
		),
		edited,
		nil,
	)

	if err = remote.PullQueryGroupFromRemote(
		edited,
		pushQueryGroup,
		repo.ImporterOptions{}.WithPrintCopies(true),
	); err != nil {
		// Preserve the temp workspace so the edit is not lost (FDR-0023).
		req.Cancel(
			errors.Wrapf(
				err,
				"ephemeral push failed; workspace kept at %s",
				tempDir,
			),
		)
		return
	}

	// Teardown only on success.
	if err = os.Chdir(originalCwd); err != nil {
		req.Cancel(err)
		return
	}

	if err = os.RemoveAll(tempDir); err != nil {
		req.Cancel(err)
		return
	}
}
