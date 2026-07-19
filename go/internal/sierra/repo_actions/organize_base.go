package repo_actions

import (
	"bytes"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/0/hyphence"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/type_blobs"
	"code.linenisgreat.com/dodder/go/internal/hotel/import_plan"
	"code.linenisgreat.com/dodder/go/internal/kilo/orgie"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
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

// writeBareBlob writes content to the repo's default blob store with no
// owning object (dodder#374(b) plan §3, OQ2: bare, collectable, no
// owning object -- writeBlobContent in update_object.go is the
// object-coupled sibling of this).
func writeBareBlob(
	repo *local_working_copy.Repo,
	content []byte,
) (digest mad_domain_interfaces.MarklId, err error) {
	blobWriter, err := repo.GetEnvRepo().GetDefaultBlobStore().MakeBlobWriter(nil)
	if err != nil {
		err = errors.Wrap(err)
		return digest, err
	}

	defer errors.DeferredCloser(&err, blobWriter)

	if _, err = bytes.NewReader(content).WriteTo(blobWriter); err != nil {
		err = errors.Wrap(err)
		return digest, err
	}

	digest = blobWriter.GetMarklId()

	return digest, err
}

// WriteOrganizeBaseAndActivate implements the dodder#374(b) plan §3
// generation sequence (per the 2026-07-18 review's excise-and-parse
// ruling), with no circularity by construction:
//
//  1. ot's Metadata has no `_base` OptionComment active yet (only the
//     prototype is registered, from MakeOptionCommentSet) -- render the
//     complete outer document text as-is; it has no `_base` line.
//  2. Wrap that rendered text as the base blob's BODY, under an envelope
//     whose metadata carries only `_group-by="..."` (present iff
//     groupingTags is non-empty) and `! organize-base-v1` last. Write
//     the envelope+body as a bare blob (writeBareBlob), obtain its
//     digest.
//  3. Activate `_base=@<digest>` on ot's Metadata (AddPrototypeAndOption,
//     overwriting the empty prototype with the real value) so the next
//     WriteTo -- the one the caller actually presents/embeds -- includes
//     it.
//
// Apply-side symmetry (plan §4): the same excision happens to `patch`
// before diffing against the base blob's body, so both sides of the
// diff are parsed by the exact same Text.ReadFrom, never a bespoke
// internal format.
// Deliberately does NOT call EnsureOrganizeBaseType -- per the
// 2026-07-18 ruling, materializing the !organize-base-v1 type object is
// a write-path concern (interactive/commit-directly, which already
// print commit confirmations and so pay for EnsureOrganizeBaseType's
// one-time creation print consistently) rather than a shared-generation
// concern. output-only writes the bare blob and never needs the type
// object to exist -- the envelope's `! organize-base-v1` line is a
// self-describing string, not a materialized reference anything
// dereferences. Callers on the write-path modes call
// EnsureOrganizeBaseType themselves before committing.
func WriteOrganizeBaseAndActivate(
	repo *local_working_copy.Repo,
	ot *orgie.Text,
	groupingTags ids.TagSlice,
) (err error) {
	var bodyBuf bytes.Buffer

	if _, err = ot.WriteTo(&bodyBuf); err != nil {
		err = errors.Wrap(err)
		return err
	}

	envelopeMetadata := orgie.NewMetadata(ids.RepoId{})

	if err = envelopeMetadata.Type.Set(OrganizeBaseTypeString); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if groupingTags.Len() > 0 {
		values := make([]string, 0, groupingTags.Len())

		for _, tag := range groupingTags {
			values = append(values, tag.String())
		}

		envelopeMetadata.OptionCommentSet.AddPrototypeAndOption(
			"group-by",
			&orgie.OptionCommentGroupBy{Value: strings.Join(values, ",")},
		)
	}

	var envelopeMetaBuf bytes.Buffer

	if _, err = envelopeMetadata.WriteTo(&envelopeMetaBuf); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var fullBuf bytes.Buffer

	hw := hyphence.Writer{
		Metadata: &envelopeMetaBuf,
		Blob:     &bodyBuf,
	}

	if _, err = hw.WriteTo(&fullBuf); err != nil {
		err = errors.Wrap(err)
		return err
	}

	digest, err := writeBareBlob(repo, fullBuf.Bytes())
	if err != nil {
		err = errors.Wrap(err)
		return err
	}

	baseDigest := &orgie.OptionCommentBaseDigest{}

	if err = baseDigest.Id.Set("@" + digest.String()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	ot.Metadata.OptionCommentSet.AddPrototypeAndOption("base", baseDigest)

	return err
}
