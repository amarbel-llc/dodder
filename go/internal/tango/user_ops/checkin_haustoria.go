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

		object, err := op.makeObjectFromDecompileResult(result)
		if err != nil {
			return nil, errors.Wrapf(err,
				"make object from %s", resource.ExternalId,
			)
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

func (op CheckinHaustoria) makeObjectFromDecompileResult(
	result haustoria.DecompileResult,
) (object *sku.Transacted, err error) {
	object, _ = sku.GetTransactedPool().GetWithRepool() //repool:owned

	metadata := object.GetMetadataMutable()

	if err = metadata.GetDescriptionMutable().Set(result.Description); err != nil {
		err = errors.Wrap(err)
		return nil, err
	}

	if result.TypeId != "" {
		if err = metadata.GetTypeMutable().SetType(result.TypeId); err != nil {
			err = errors.Wrap(err)
			return nil, err
		}
	}

	for _, tagStr := range result.Tags {
		if err = metadata.AddTagString(tagStr); err != nil {
			err = errors.Wrap(err)
			return nil, err
		}
	}

	if len(result.Blob) > 0 {
		if err = op.writeBlob(object, result.Blob); err != nil {
			return nil, err
		}
	}

	return object, nil
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
