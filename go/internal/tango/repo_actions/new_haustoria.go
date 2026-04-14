package repo_actions

import (
	"os"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/charlie/haustoria"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/object_metadata_fmt_hyphence"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/india/import_plan"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/delta/files"
)

type NewHaustoria struct {
	*repo
	Haustoria  haustoria.Haustoria
	TextParser object_metadata_fmt_hyphence.Parser
	Proto      sku.Proto
}

func (op NewHaustoria) Run(
	args ...string,
) (results sku.TransactedMutableSet, err error) {
	builder := import_plan.MakeLocalBuilder()

	builder.AddTransform(
		import_plan.MakeAllocateZettelIdTransform(
			op.GetStore().GetZettelIdIndex(),
		),
	)

	for _, arg := range args {
		var reader *os.File

		if arg == "-" {
			reader = os.Stdin
		} else {
			if reader, err = os.Open(arg); err != nil {
				return nil, errors.Wrapf(err, "open %s", arg)
			}
			defer files.CloseReadOnly(reader)
		}

		object, _ := sku.GetTransactedPool().GetWithRepool() //repool:owned

		if _, err = op.TextParser.ParseMetadata(reader, object); err != nil {
			return nil, errors.Wrapf(err, "parse %s", arg)
		}

		op.Proto.Apply(object, genres.Zettel)

		if err = builder.AddObject(object, 0); err != nil {
			return nil, errors.Wrap(err)
		}
	}

	plan, buildErr := builder.Build()
	if buildErr != nil {
		return nil, errors.Wrap(buildErr)
	}

	plan.DefaultCommitOptions = sku.CommitOptions{
		Proto: op.GetStore().GetProtoZettel(),
		StoreOptions: sku.StoreOptions{
			LockfileOptions: sku.LockfileOptions{
				AllowTagFailure: true,
			},
			AddToInventoryList: true,
			UpdateTai:          true,
			RunHooks:           true,
			Validate:           true,
			ApplyProto:         true,
		},
	}

	results, err = op.repo.ExecutePlan(plan)
	if err != nil {
		return results, err
	}

	for object := range results.All() {
		if _, decompileErr := op.decompileObject(object); decompileErr != nil {
			return results, errors.Wrapf(decompileErr,
				"decompile %s", object.GetObjectId(),
			)
		}
	}

	return results, err
}

func (op NewHaustoria) decompileObject(
	object *sku.Transacted,
) (haustoria.DecompileResult, error) {
	var tags []string
	for tag := range object.GetMetadata().GetTags().All() {
		tags = append(tags, tag.String())
	}

	return op.Haustoria.Decompile(haustoria.DecompileRequest{
		Description: object.GetMetadata().GetDescription().String(),
		Tags:        tags,
		TypeId:      object.GetMetadata().GetType().String(),
	})
}
