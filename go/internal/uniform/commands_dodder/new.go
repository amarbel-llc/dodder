package commands_dodder

import (
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/0/checkout_mode"
	"code.linenisgreat.com/dodder/go/internal/0/haustoria"
	"code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/repo_id"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/echo/object_metadata_fmt_hyphence"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/orgie"
	"code.linenisgreat.com/dodder/go/internal/lima/store_fs"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/sierra/repo_actions"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/dodder/go/lib/bravo/script_value"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

func init() {
	utility.AddCmd("new", &New{})
}

func (cmd New) GetDescription() command.Description {
	return command.Description{
		Short: "create new zettels",
		Long: "Create one or more new zettels in the store. With no " +
			"arguments, creates a single empty zettel and opens it " +
			"for editing. File path arguments import existing files " +
			"as zettel blobs. Use -count to create multiple empty " +
			"zettels, -description, -tags, and -type to set metadata, " +
			"and -shas to attach blobs already in the store. Use " +
			"-object-id to assign a chosen id to a single new object " +
			"(a zettel like 'a/b', a tag like 'foo', or a type like " +
			"'!task'); an empty -object-id auto-assigns a zettel id. A " +
			"non-zettel -object-id sets the meta-type automatically from " +
			"the id's genre. Use -blob to write an inline blob body for " +
			"that single object.",
	}
}

type New struct {
	command_components_dodder.LocalWorkingCopy

	complete command_components_dodder.Complete

	ids.RepoId
	Count int
	// TODO combine organize and edit and refactor
	command_components_dodder.Checkout
	PrintOnly bool
	Filter    script_value.ScriptValue
	Shas      bool

	// ObjectId, when non-empty, assigns a caller-chosen object-id to a single
	// new object (a zettel `a/b`, a tag `foo`, or a type `!task`). Empty keeps
	// the default behavior of an auto-assigned zettel id.
	ObjectId string
	// Blob, when non-empty, is written verbatim as the new object's blob body.
	// Only valid on the no-positional-args path. Empty writes no blob.
	Blob string

	// Ephemeral, when set, creates the new object(s) against a temp
	// repo-backed workspace pulled from a resolved parent repo, pushes them
	// back, and tears the temp workspace down — no persistent .dodder-workspace
	// required (FDR-0023). ParentPath is the explicit -parent path; empty means
	// the home repo.
	Ephemeral  bool
	ParentPath string

	ephemeral command_components_dodder.ParentBackedWorkspace

	sku.Proto
}

var (
	_ interfaces.CommandComponentWriter = (*New)(nil)
	_ command.CommandWithResetCLIState  = (*New)(nil)
)

// ResetCLIState clears the flag-bound state that accumulates across
// invocations when this registered command value is reused in a
// long-lived process (the MCP bridge): descriptions.Description.Set
// appends and the tag set unions, so without this two MCP `new` calls
// concatenate descriptions (#247). Scalar flags self-heal via the
// defaults written at flag registration; only the Var-bound Proto
// metadata needs explicit clearing.
func (cmd *New) ResetCLIState() {
	objects.Resetter.Reset(&cmd.Proto.Metadata)
}

func (cmd *New) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{{
		Args: []command.Arg{{
			Name:        "shas",
			Description: "blob SHAs to create zettels from (requires -shas flag)",
			Variadic:    true,
		}},
	}}
}

func (cmd *New) SetFlagDefinitions(flagSet interfaces.CLIFlagDefinitions) {
	flagSet.Var(&cmd.RepoId, "kasten", "none or Browser")

	flagSet.BoolVar(
		&cmd.Shas,
		"shas",
		false,
		"treat arguments as blobs that are already checked in",
	)

	flagSet.IntVar(
		&cmd.Count,
		"count",
		1,
		"when creating new empty zettels, how many to create. otherwise ignored",
	)

	flagSet.Var(
		&cmd.Filter,
		"filter",
		"a script to run for each file to transform it the standard zettel format",
	)

	flagSet.StringVar(
		&cmd.ObjectId,
		"object-id",
		"",
		"object id to assign to a single new object: a zettel like 'a/b', a tag "+
			"like 'foo', or a type like '!task'. Empty (default) auto-assigns a "+
			"zettel id. Mutually exclusive with -count>1 and positional args.",
	)

	flagSet.StringVar(
		&cmd.Blob,
		"blob",
		"",
		"inline blob body for a single new object (no-arg path only). Empty "+
			"(default) writes no blob.",
	)

	cmd.complete.SetFlagsProto(
		&cmd.Proto,
		flagSet,
		"description to use for new zettels",
		"tags added for new zettels",
		"type used for new zettels",
	)

	flagSet.BoolVar(
		&cmd.Ephemeral,
		"ephemeral",
		false,
		"create the new object(s) against a temp repo-backed workspace pulled from a resolved parent repo, then push them back and tear the temp workspace down (no persistent .dodder-workspace required)",
	)

	flagSet.StringVar(
		&cmd.ParentPath,
		"parent",
		"",
		"path to a CWD-scoped parent dodder repository for -ephemeral (omit for home repo)",
	)

	cmd.Checkout.SetFlagDefinitions(flagSet)
}

func (cmd New) runHaustoria(
	repo *local_working_copy.Repo,
	h haustoria.Haustoria,
	format object_metadata_fmt_hyphence.Format,
	args []string,
) sku.TransactedMutableSet {
	if len(args) == 0 {
		emptyOp := repo_actions.MakeWriteNewZettels(repo)

		objects, err := emptyOp.RunMany(cmd.Proto, cmd.Count)
		if err != nil {
			repo.Cancel(err)
		}

		// Decompile empty objects to external store
		for object := range objects.All() {
			var tags []string
			for tag := range object.GetMetadata().GetTags().All() {
				tags = append(tags, tag.String())
			}

			if _, err := h.Decompile(haustoria.DecompileRequest{
				Description: object.GetMetadata().GetDescription().String(),
				Tags:        tags,
				TypeId:      object.GetMetadata().GetType().String(),
			}); err != nil {
				repo.Cancel(err)
			}
		}

		return objects
	}

	op := repo_actions.MakeNewHaustoria(repo, h, format, cmd.Proto)

	objects, err := op.Run(args...)
	if err != nil {
		repo.Cancel(err)
	}

	return objects
}

func (cmd New) ValidateFlagsAndArgs(
	repo *local_working_copy.Repo,
	args ...string,
) (err error) {
	if repo.GetConfig().IsDryRun() && len(args) == 0 {
		err = errors.ErrorWithStackf(
			"when -dry-run is set, paths to existing zettels must be provided",
		)
		return err
	}

	// -object-id / -blob name exactly one new object with an inline body, so
	// they only make sense on the single, no-positional-args path.
	if cmd.ObjectId != "" || cmd.Blob != "" {
		if len(args) > 0 {
			err = errors.BadRequestf(
				"-object-id / -blob cannot be combined with positional arguments",
			)
			return err
		}

		if cmd.Count > 1 {
			err = errors.BadRequestf(
				"-object-id / -blob cannot be combined with -count > 1",
			)
			return err
		}

		if _, ok := repo.GetEnvWorkspace().GetStore().StoreLike.(haustoria.Haustoria); ok {
			err = errors.BadRequestf(
				"-object-id / -blob are not supported against a haustoria workspace store",
			)
			return err
		}
	}

	// A chosen non-Zettel object-id (a type or tag) forces its meta-type to the
	// genre default (e.g. !toml-type-v2); an explicit -type would be silently
	// overridden, so reject the combination rather than mislead.
	if cmd.ObjectId != "" && !ids.IsEmpty(cmd.Proto.Metadata.GetType()) {
		objectId, repool, parseErr := ids.MakeObjectId(cmd.ObjectId)
		if parseErr != nil {
			err = errors.BadRequestf(
				"invalid -object-id %q: %s", cmd.ObjectId, parseErr,
			)
			return err
		}

		genre := genres.Make(objectId.GetGenre())
		repool()

		if genre != genres.Zettel {
			err = errors.BadRequestf(
				"-type cannot be combined with a non-zettel -object-id (%s); "+
					"the meta-type is set automatically from the id's genre",
				cmd.ObjectId,
			)
			return err
		}
	}

	// #290: in a repo with no default type (ExcludeDefaultType — every
	// workspace repo and clone), the from-scratch creation path with no -type
	// would commit a typeless object whose bare `!` later breaks push/import
	// with `unsupported seq: "!"`. Reject up front. This only applies to the
	// no-positional-args path (path/sha inputs carry their type in the blob
	// header); a non-zettel -object-id gets its meta-type from the genre, so
	// only zettel-genre results need a type.
	if len(args) == 0 && ids.IsEmpty(cmd.Proto.Metadata.GetType()) {
		if _, ok := repo.GetEnvWorkspace().GetStore().StoreLike.(haustoria.Haustoria); !ok {
			resultGenre := genres.Zettel

			if cmd.ObjectId != "" {
				objectId, repool, parseErr := ids.MakeObjectId(cmd.ObjectId)
				if parseErr != nil {
					err = errors.BadRequestf(
						"invalid -object-id %q: %s", cmd.ObjectId, parseErr,
					)
					return err
				}

				resultGenre = genres.Make(objectId.GetGenre())
				repool()
			}

			if resultGenre == genres.Zettel &&
				repo.GetEnvWorkspace().GetDefaults().GetDefaultType().IsEmpty() {
				err = errors.BadRequestf(
					"no type given and repo has no default type; pass -type",
				)
				return err
			}
		}
	}

	return err
}

func (cmd *New) Run(req command.Request) {
	if cmd.Ephemeral {
		cmd.runEphemeral(req)
		return
	}

	cmd.runInWorkspace(req, req.PopArgs())
}

// runInWorkspace is the normal `new` flow: create the object(s) in the current
// workspace/repo. Also invoked by runEphemeral after it has materialized and
// chdir'd into a temp repo-backed workspace. Args are passed in (rather than
// popped here) so runEphemeral can resolve positional file paths to absolute
// BEFORE the chdir — a relative path would otherwise resolve against the temp
// dir, not the caller's cwd.
func (cmd *New) runInWorkspace(req command.Request, args []string) {
	repo := cmd.MakeLocalWorkingCopy(req)

	if err := cmd.ValidateFlagsAndArgs(repo, args...); err != nil {
		repo.Cancel(err)
	}

	textFormatterOptions := checkout_options.TextFormatterOptions{}

	format := object_metadata_fmt_hyphence.Factory{
		EnvDir:    repo.GetEnvRepo(),
		BlobStore: repo.GetEnvRepo().GetDefaultBlobStore(),
	}.Make()

	var objects sku.TransactedMutableSet

	if h, ok := repo.GetEnvWorkspace().GetStore().StoreLike.(haustoria.Haustoria); ok {
		objects = cmd.runHaustoria(repo, h, format, args)
	} else if len(args) == 0 {
		emptyOp := repo_actions.MakeWriteNewZettels(repo)

		if cmd.ObjectId != "" || cmd.Blob != "" {
			objectId, repool, err := ids.MakeObjectId(cmd.ObjectId)
			if err != nil {
				repo.Cancel(errors.BadRequestf(
					"invalid -object-id %q: %s", cmd.ObjectId, err,
				))
			}

			defer repool()

			object, err := emptyOp.RunOneWithObjectId(
				cmd.Proto,
				objectId,
				cmd.Blob,
			)
			if err != nil {
				repo.Cancel(err)
			}

			objects = sku.MakeTransactedMutableSet()
			if err := objects.Add(object); err != nil {
				repo.Cancel(err)
			}
		} else {
			var err error

			if objects, err = emptyOp.RunMany(cmd.Proto, cmd.Count); err != nil {
				repo.Cancel(err)
			}
		}
	} else if cmd.Shas {
		opCreateFromShas := repo_actions.MakeCreateFromShas(repo)
		opCreateFromShas.Proto = cmd.Proto

		{
			var err error

			if objects, err = opCreateFromShas.Run(args...); err != nil {
				repo.Cancel(err)
			}
		}
	} else {
		opCreateFromPath := repo_actions.MakeCreateFromPaths(repo, format)
		opCreateFromPath.Filter = cmd.Filter
		opCreateFromPath.Delete = cmd.Delete
		opCreateFromPath.Proto = cmd.Proto

		{
			var err error

			if objects, err = opCreateFromPath.Run(args...); err != nil {
				if errors.IsNotExist(err) {
					errors.ContextCancelWithBadRequestf(repo, "Expected a valid file path. Did you mean to add `-description`?")
				} else {
					repo.Cancel(err)
				}
			}
		}
	}

	// TODO make mutually exclusive with organize
	if cmd.Edit {
		opCheckout := repo_actions.MakeCheckout(repo)
		opCheckout.Options = checkout_options.Options{
			CheckoutMode: checkout_mode.Make(checkout_mode.MetadataAndBlob),
			OptionsWithoutMode: checkout_options.OptionsWithoutMode{
				StoreSpecificOptions: store_fs.CheckoutOptions{
					ForceInlineBlob:      true,
					TextFormatterOptions: textFormatterOptions,
				},
			},
		}
		opCheckout.Edit = true
		opCheckout.RefreshCheckout = true

		if _, err := opCheckout.Run(objects); err != nil {
			repo.Cancel(err)
		}
	}

	if cmd.Organize {
		// orgie.Metadata's embedded OptionCommentSet must be built via
		// MakeOptionCommentSet — a zero value leaves its prototype map nil and
		// GetOptions panics ("Metadata not initalized"). Every other organize
		// caller (last.go, clean.go, checkin.go) does this; #345.
		opOrganize := repo_actions.MakeOrganize(
			repo,
			orgie.Metadata{
				OptionCommentSet: orgie.MakeOptionCommentSet(nil),
			},
		)

		if err := opOrganize.Metadata.SetFromObjectMetadata(
			&cmd.Metadata,
			ids.RepoId{},
		); err != nil {
			repo.Cancel(err)
		}

		var results orgie.OrganizeResults

		{
			var err error

			if results, err = opOrganize.RunWithTransacted(nil, objects); err != nil {
				repo.Cancel(err)
			}
		}

		if _, err := repo_actions.LockAndCommitOrganizeResults(
			repo,
			results,
		); err != nil {
			repo.Cancel(err)
		}
	}
}

// runEphemeral implements FDR-0023 for `new`: it materializes a temp
// repo-backed workspace whose blob store points at a resolved parent repo,
// creates the new object(s) there (via runInWorkspace), pushes them back to
// the parent, then tears the temp workspace down. On failure the temp
// workspace is preserved and its path surfaced so no work is lost. Mirrors
// Edit.runEphemeral.
func (cmd *New) runEphemeral(req command.Request) {
	// Workspace repos are always CWD-rooted; here the CWD is the temp dir.
	// Mutate the SHARED *repo_config_cli.Config (not a FromAny copy) so
	// Genesis.OnTheFirstDay — which reads config via its own FromAny — sees the
	// CWD-scoped RepoId. A FromAny copy would leave genesis reading the original
	// auto/unknown-location id, which resolves to the XDG home fallback and
	// roots .dodder/.madder (and the pointer blob store) OUTSIDE the temp dir.
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
	// its name before config.RepoId is overwritten with CwdDefault below.
	if !repo_id.IsAuto(config.RepoId) {
		cmd.ephemeral.ParentRepoId = config.RepoId.GetName()
	}

	config.RepoId = repo_id.CwdDefault()
	cmd.ephemeral.ParentPath = cmd.ParentPath

	// Resolve positional file-path arguments to absolute BEFORE RunEphemeral
	// chdirs into the temp workspace. Positional args to `new` are file paths to
	// import (the Move-to-Dodder / -organize path); a relative path would resolve
	// against the temp dir after the chdir and fail to be found. -shas args are
	// blob SHAs, not paths, so leave them verbatim. This is a local fix; #347
	// tracks hoisting path-resolution into RunEphemeral as free machinery for all
	// commands, and #346 tracks presenting the literal arg while manipulating the
	// absolute one.
	args := req.PopArgs()

	if !cmd.Shas {
		for i, arg := range args {
			if abs, absErr := filepath.Abs(arg); absErr == nil {
				args[i] = abs
			}
		}
	}

	// The shared lifecycle (temp repo-backed workspace, pull, push, teardown)
	// lives in ParentBackedWorkspace.RunEphemeral; the new-specific step is
	// creating the object(s) via the normal in-workspace flow. runInWorkspace
	// opens its own working copy of the temp workspace (resolved as writable,
	// pointer store discovered from tempDir) and handles its own errors via
	// req.Cancel, so the closure ignores the passed working copy and returns
	// nil.
	cmd.ephemeral.RunEphemeral(
		req,
		nil,
		func(_ *local_working_copy.Repo) error {
			cmd.runInWorkspace(req, args)
			return nil
		},
	)
}
