package repo_actions

import (
	"bytes"

	"code.linenisgreat.com/dodder/go/internal/0/haustoria"
	"code.linenisgreat.com/dodder/go/internal/bravo/checked_out_state"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/import_plan"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/kilo/store_workspace"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
)

type CheckinHaustoria struct {
	*repo
	Haustoria haustoria.Haustoria
	StoreLike store_workspace.StoreLike
	Query     *queries.Query
}

func (op CheckinHaustoria) Run() (results sku.TransactedMutableSet, err error) {
	// Use QueryCheckedOut which does single-fetch per calendar and applies
	// query filtering. This avoids the N+1 problem of Discover + per-resource
	// Compile.
	var checkedOut []sku.SkuType

	if err = op.StoreLike.QueryCheckedOut(
		op.Query,
		func(co sku.SkuType) error {
			cloned, _ := co.Clone() //repool:owned
			checkedOut = append(checkedOut, cloned)
			return nil
		},
	); err != nil {
		return nil, errors.Wrap(err)
	}

	if len(checkedOut) == 0 {
		return sku.MakeTransactedMutableSet(), nil
	}

	zettelIdIndex := op.GetStore().GetZettelIdIndex()
	builder := import_plan.MakeLocalBuilder()

	builder.AddTransform(
		import_plan.MakeAllocateZettelIdTransform(zettelIdIndex),
	)

	for _, co := range checkedOut {
		// Skip already-bound resources — only create objects for new ones.
		if co.GetState() != checked_out_state.Untracked {
			continue
		}

		external := co.GetSkuExternal()

		proto := op.makeProtoFromExternal(external)
		object, _ := proto.Make() //repool:owned

		blobDigest := external.GetMetadata().GetBlobDigest()
		if !blobDigest.IsNull() {
			if err = object.SetBlobDigest(blobDigest); err != nil {
				return nil, errors.Wrap(err)
			}
		}

		// Bind the dodder object to the CalDAV UID so subsequent queries
		// can look up the binding and avoid creating duplicates.
		object.GetExternalObjectIdMutable().ResetWith(
			external.GetExternalObjectIdMutable(),
		)

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
			AddToInventoryList: true,
			UpdateTai:          true,
			RunHooks:           true,
			Validate:           true,
			ApplyProto:         true,
		},
	}

	results, err = op.repo.ExecutePlan(plan)

	return results, err
}

func (op CheckinHaustoria) makeProtoFromExternal(
	external *sku.Transacted,
) sku.Proto {
	proto := sku.MakeProto(nil)

	proto.Metadata.GetDescriptionMutable().Set(
		external.GetMetadata().GetDescription().String(),
	)

	typeStr := external.GetMetadata().GetType().String()
	if typeStr != "" {
		proto.Metadata.GetTypeMutable().SetType(typeStr)
	}

	for tag := range external.GetMetadata().GetTags().All() {
		proto.Metadata.AddTagPtr(tag)
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
