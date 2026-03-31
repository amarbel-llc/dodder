package user_ops

import (
	"bytes"

	"code.linenisgreat.com/dodder/go/internal/charlie/haustoria"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/india/import_plan"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type CheckinHaustoria struct {
	*local_working_copy.Repo
	Haustoria haustoria.Haustoria
}

func (op CheckinHaustoria) Run() (results sku.TransactedMutableSet, err error) {
	resources, err := op.Haustoria.Discover()
	if err != nil {
		return nil, errors.Wrap(err)
	}

	if len(resources) == 0 {
		return sku.MakeTransactedMutableSet(), nil
	}

	zettelIdIndex := op.GetStore().GetZettelIdIndex()
	builder := import_plan.MakeLocalBuilder()

	builder.AddTransform(
		import_plan.MakeAllocateZettelIdTransform(zettelIdIndex),
	)

	for _, resource := range resources {
		result, decompileErr := op.Haustoria.Decompile(
			haustoria.DecompileRequest{
				ExternalId: resource.ExternalId,
			},
		)
		if decompileErr != nil {
			return nil, errors.Wrapf(decompileErr,
				"decompile %s", resource.ExternalId,
			)
		}

		proto := op.makeProtoFromDecompileResult(result)
		object, _ := proto.Make() //repool:owned

		if len(result.Blob) > 0 {
			if err = op.writeBlob(object, result.Blob); err != nil {
				return nil, errors.Wrapf(err,
					"write blob for %s", resource.ExternalId,
				)
			}
		}

		builder.AddObject(object, 0)
	}

	plan, buildErr := builder.Build()
	if buildErr != nil {
		return nil, errors.Wrap(buildErr)
	}

	plan.DefaultCommitOptions = sku.CommitOptions{
		Proto: op.GetStore().GetProtoZettel(),
		StoreOptions: sku.StoreOptions{
			AddToInventoryList: true,
			UpdateTai:          true,
			RunHooks:           true,
			Validate:           true,
			ApplyProto:         true,
		},
	}

	results, err = op.Repo.ExecutePlan(plan)

	return results, err
}

func (op CheckinHaustoria) makeProtoFromDecompileResult(
	result haustoria.DecompileResult,
) sku.Proto {
	proto := sku.MakeProto(nil)

	proto.Metadata.GetDescriptionMutable().Set(result.Description)

	if result.TypeId != "" {
		proto.Metadata.GetTypeMutable().SetType(result.TypeId)
	}

	for _, tagStr := range result.Tags {
		proto.Metadata.AddTagString(tagStr)
	}

	return proto
}

func (op CheckinHaustoria) writeBlob(
	object *sku.Transacted,
	content []byte,
) (err error) {
	blobWriter, err := op.GetEnvRepo().GetDefaultBlobStore().MakeBlobWriter(nil)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	defer errors.DeferredCloser(&err, blobWriter)

	if _, err = bytes.NewReader(content).WriteTo(blobWriter); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = object.SetBlobDigest(blobWriter.GetMarklId()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
