package repo_actions

import (
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/type_blobs"
	"code.linenisgreat.com/dodder/go/internal/hotel/import_plan"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

// OrganizeBaseTypeString is the type string for the user-space
// organize-base-v1 type (dodder#374(b) plan §2) -- a bare, opaque blob
// type recording an organize session's generated ground form. Never a
// builtin (docs/rfcs/0003-cutting-garden-receipt-ingest.md:85-86); the
// definition here mirrors genesis's !md/!task pattern
// (local_working_copy/genesis.go:195-246) but runs lazily from organize
// itself rather than at genesis time.
const OrganizeBaseTypeString = "organize-base-v1"

// EnsureOrganizeBaseType lazily and idempotently creates the
// !organize-base-v1 type object if it doesn't already exist (plan §2,
// OQ1: lazy creation, tolerant of a concurrent creator -- "already
// exists" is success, not an error).
func EnsureOrganizeBaseType(repo *local_working_copy.Repo) (err error) {
	objectIdType := ids.MustTypeStruct(OrganizeBaseTypeString)

	if _, err = repo.GetStore().ReadOneObjectId(objectIdType); err == nil {
		return nil
	} else if !errors.IsErrNotFound(err) {
		err = errors.Wrap(err)
		return err
	}

	err = nil

	tipe := ids.DefaultOrPanic(genres.Type)

	// The base blob's content is the serialized organize/espalier form
	// (plan §9), not TOML -- amended per review from the original
	// toml-for-non-toml draft.
	blob := type_blobs.TomlV2{
		FileExtension: "organize",
		VimSyntaxType: "markdown",
	}

	object, objectRepool := sku.GetTransactedPool().GetWithRepool() //repool:owned
	defer objectRepool()

	if err = object.GetObjectIdMutable().SetWithId(objectIdType); err != nil {
		err = errors.Wrap(err)
		return err
	}

	digest, _, err := repo.GetStore().GetTypedBlobStore().Type.SaveBlobText(
		tipe,
		&blob,
	)
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	object.GetMetadataMutable().GetBlobDigestMutable().ResetWithMarklId(digest)
	object.GetMetadataMutable().GetTypeMutable().ResetWithType(tipe)

	builder := import_plan.MakeLocalBuilder()

	if err = builder.AddObject(object, 0); err != nil {
		err = errors.Wrap(err)
		return err
	}

	plan, buildErr := builder.Build()
	if buildErr != nil {
		err = errors.Wrap(buildErr)
		return err
	}

	plan.DefaultCommitOptions = sku.CommitOptions{
		StoreOptions: sku.StoreOptions{
			AddToInventoryList: true,
			UpdateTai:          true,
			RunHooks:           true,
			Validate:           true,
		},
	}

	if _, err = repo.ExecutePlan(plan); err != nil {
		// Tolerate a concurrent creator: if the type now exists (another
		// organize invocation raced us and won), treat this as success
		// rather than surfacing the commit conflict (plan §2's
		// idempotent-not-racy requirement).
		if _, readErr := repo.GetStore().ReadOneObjectId(objectIdType); readErr == nil {
			return nil
		}

		err = errors.Wrap(err)
		return err
	}

	return err
}
