package commands_dodder

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/import_plan"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/tango/command_components_dodder"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

func init() {
	// FDR-0024 / RFC-0008: the list-in/list-out Lua transform over an
	// expanded inventory list. Supersedes the deleted
	// prototype-lua-transform command (Forgejo #370 item 1). See
	// docs/features/0024-inventory-list-transform-plugins.md and
	// docs/rfcs/0008-inventory-list-transform-plugin-api.md.
	//
	// The `transform` command is the query-source consumer of the shared
	// transform pipeline (transform_pipeline.go); dodder#392 hangs the `init`
	// (inventory-list union) and `clone -script` consumers off the same
	// pipeline.
	utility.AddCmd("transform", &Transform{})
}

type Transform struct {
	command_components_dodder.LocalWorkingCopyWithQueryGroup

	Script         string
	ScriptDigest   string
	DryRun         bool
	SkipValidation bool
	NoNewObjects   bool
}

var (
	_ interfaces.CommandComponentWriter = (*Transform)(nil)
	_ command.CommandWithArgs           = (*Transform)(nil)
)

func (cmd *Transform) GetArgs() []command.ArgGroup {
	return []command.ArgGroup{cmd.Query.GetArgGroup()}
}

func (cmd Transform) GetDescription() command.Description {
	return command.Description{
		Short: "run a Lua list-in/list-out transform over queried objects and commit the result",
	}
}

func (cmd *Transform) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	cmd.LocalWorkingCopyWithQueryGroup.SetFlagDefinitions(flagSet)

	flagSet.StringVar(
		&cmd.Script,
		"script",
		"",
		"path to the Lua transform script (mutually exclusive with -script-digest)",
	)

	flagSet.StringVar(
		&cmd.ScriptDigest,
		"script-digest",
		"",
		"markl id of a stored blob containing the Lua transform script (mutually exclusive with -script)",
	)

	flagSet.BoolVar(
		&cmd.DryRun,
		"dry_run",
		false,
		"build and validate the output plan and report it without committing",
	)

	flagSet.BoolVar(
		&cmd.SkipValidation,
		"skip_validation",
		false,
		"skip the fsck-style validation of the transform output (for staged, intentionally-inconsistent migration passes)",
	)

	flagSet.BoolVar(
		&cmd.NoNewObjects,
		"no_new_objects",
		false,
		"reject any output object whose object id is not present in the input list",
	)
}

func (cmd Transform) Run(req command.Request) {
	localWorkingCopy, queryGroup := cmd.MakeLocalWorkingCopyAndQueryGroup(
		req,
		queries.BuilderOptions(
			queries.BuilderOptionDefaultSigil(
				ids.SigilLatest,
				ids.SigilHidden,
			),
			queries.BuilderOptionDefaultGenres(genres.Zettel),
		),
	)

	scriptReader, err := makeTransformScriptReader(
		localWorkingCopy,
		cmd.Script,
		cmd.ScriptDigest,
	)
	if err != nil {
		localWorkingCopy.Cancel(err)
		return
	}

	defer errors.ContextMustClose(localWorkingCopy, scriptReader)

	list, skippedEdges, err := localWorkingCopy.MakeExpandedInventoryList(
		queryGroup,
	)
	if err != nil {
		localWorkingCopy.Cancel(errors.Wrap(err))
		return
	}

	if len(skippedEdges) > 0 {
		// A mid-migration repo (the -skip_validation use case) may hold
		// dangling references that make expansion partially fail; refusing
		// to open it would deadlock the staged-migration workflow the flag
		// exists for.
		if cmd.SkipValidation {
			localWorkingCopy.GetUI().Printf(
				"warning: edge traversal had %d failure(s); continuing due to -skip_validation",
				len(skippedEdges),
			)
		} else {
			errors.ContextCancelWithErrorf(
				localWorkingCopy,
				"edge traversal had %d failure(s): %s",
				len(skippedEdges),
				skippedEdges[0],
			)
			return
		}
	}

	var objects []*sku.Transacted

	for object := range list.All() {
		cloned, _ := object.CloneTransacted() //repool:owned
		objects = append(objects, cloned)
	}

	localWorkingCopy.GetUI().Printf("selected %d object(s)", len(objects))

	pipeline := transformPipeline{
		repo:           localWorkingCopy,
		scriptReader:   scriptReader,
		objects:        objects,
		dryRun:         cmd.DryRun,
		skipValidation: cmd.SkipValidation,
		noNewObjects:   cmd.NoNewObjects,
		// transform's query source yields one latest version per id, so a
		// same-id output means the script merged two objects — reject it.
		disallowDuplicateObjectIds: true,
		// transform's objects are locally-authored; ExecutePlan seals them
		// under this repo's key at the working-list flush (no re-sign needed).
		commit: func(plan *import_plan.Plan) (int, error) {
			results, err := localWorkingCopy.ExecutePlan(plan)
			if err != nil {
				return 0, err
			}

			return results.Len(), nil
		},
	}

	if err := pipeline.run(); err != nil {
		localWorkingCopy.Cancel(err)
		return
	}
}
