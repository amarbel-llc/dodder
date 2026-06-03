package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/0/checkout_mode"
	"code.linenisgreat.com/dodder/go/internal/0/haustoria"
	"code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
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
			"and -shas to attach blobs already in the store.",
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

	sku.Proto
}

var _ interfaces.CommandComponentWriter = (*New)(nil)

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

	cmd.complete.SetFlagsProto(
		&cmd.Proto,
		flagSet,
		"description to use for new zettels",
		"tags added for new zettels",
		"type used for new zettels",
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

	return err
}

func (cmd *New) Run(req command.Request) {
	args := req.PopArgs()
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

		{
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
		opOrganize := repo_actions.MakeOrganize(repo, orgie.Metadata{})

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
