package repo_actions

import (
	"bytes"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/0/hyphence"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/orgie"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/lib/0/collections_slice"
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
)

// OrganizeBaseTypeString is the envelope type-line value for a base
// blob's own hyphence metadata (`! organize-base-v1`, dodder#374(b)
// plan §2/§9) -- a plain, self-describing STRING inside the blob's
// bytes, never a materialized dodder type object. Per the 2026-07-18
// correction: an organize-base blob is never committed as, or
// associated with, an actual dodder object -- it is a bare blob only
// (writeBareBlob), so it never appears in `dodder show`, sync, or any
// commit-confirmation output. There is deliberately no
// EnsureOrganizeBaseType / type-registration mechanism; a prior
// version of this file had one and it was removed (dodder#374 commit
// history has the story) because materializing the type as a real
// object broke exactly the "never appears for user output" property
// this comment now states as the design invariant.
const OrganizeBaseTypeString = "organize-base-v1"

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
//  1. ot's Metadata has no `_base` Setting active yet (only the
//     prototype is registered, from MakeSettingSet) -- render the
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
//
// The base blob is ALWAYS bare (writeBareBlob, no owning object) in
// every mode -- output-only, commit-directly, and interactive alike.
// Nothing ever materializes `! organize-base-v1` as a real dodder type
// object (2026-07-18 correction): the envelope's type line is a plain
// descriptive string inside the blob's own bytes, so an
// organize-base blob never shows up in `dodder show`, sync, fsck, or
// any commit-confirmation output, in any mode.
func WriteOrganizeBaseAndActivate(
	repo *local_working_copy.Repo,
	ot *orgie.Text,
	groupingTags ids.TagSlice,
) (err error) {
	var bodyBuf bytes.Buffer

	// WriteDataPlaneTo, not WriteTo: the base blob's digest must depend
	// only on the document's data plane (`-`/`!` lines), never the
	// operational plane (`%`/`%:`) -- cutting-garden RFC 0015 (merged).
	// A document generated with or without operational directives must
	// produce the same base.
	if _, err = ot.WriteDataPlaneTo(&bodyBuf); err != nil {
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

		envelopeMetadata.SettingSet.AddPrototypeAndOption(
			"group-by",
			&orgie.SettingGroupBy{Value: strings.Join(values, ",")},
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

	baseDigest := &orgie.SettingBaseDigest{}

	// digest.String() alone, NOT "@"+digest.String() -- markl.Id.Set's
	// wire form is `[purpose@]<digest>` (splits on the first `@`), so a
	// leading `@` with nothing before it is an EMPTY PURPOSE, which the
	// underlying library now rejects, not a "bare digest" marker. The
	// `@` in `_base=@<digest>` is SettingBaseDigest's own display
	// convention (String() below), never passed through to Id.Set.
	if err = baseDigest.Id.Set(digest.String()); err != nil {
		err = errors.Wrap(err)
		return err
	}

	ot.Metadata.SettingSet.AddPrototypeAndOption("base", baseDigest)

	return err
}

// DereferenceOrganizeBase resolves patch's `_base=@<digest>` field to
// the base Text it points at (dodder#374(b) plan §3/§4): dereferences
// the blob, splits its envelope (metadata: `_group-by`,
// `! organize-base-v1`) from its body, and parses the body through the
// exact same orgie.Text.ReadFrom used for patch -- no bespoke internal
// format, per the excise-and-parse ruling. Does NOT mutate patch --
// excising `_base` from the patch happens inside
// orgie.ChangesFromResults (kilo tier, where the diff itself runs), not
// here.
//
// Fails fast, before any diff computation: ErrOrganizeBaseMissing if
// patch has no `_base` field at all (plan §8's cold-cutover), or
// ErrBaseUndereferenceable if `_base` is present but its digest can't
// be read back (plan §3).
func DereferenceOrganizeBase(
	repo *local_working_copy.Repo,
	patch *orgie.Text,
) (base *orgie.Text, wasGrouped bool, groupingTags ids.TagSlice, err error) {
	oc, found := patch.Metadata.SettingSet.GetByKey("base")
	if !found {
		err = errors.Wrap(orgie.ErrOrganizeBaseMissing{})
		return base, wasGrouped, groupingTags, err
	}

	ocwk, isKeyed := oc.(orgie.SettingWithKey)
	if !isKeyed {
		err = errors.Wrap(orgie.ErrOrganizeBaseMissing{})
		return base, wasGrouped, groupingTags, err
	}

	baseDigest, isDigest := ocwk.Setting.(*orgie.SettingBaseDigest)
	if !isDigest {
		err = errors.Wrap(orgie.ErrOrganizeBaseMissing{})
		return base, wasGrouped, groupingTags, err
	}

	digestString := baseDigest.Id.String()

	blobReader, readerErr := repo.GetEnvRepo().GetDefaultBlobStore().MakeBlobReader(
		baseDigest.Id,
	)
	if readerErr != nil {
		err = errors.Wrap(orgie.ErrBaseUndereferenceable{
			Digest: digestString,
			Cause:  readerErr,
		})
		return base, wasGrouped, groupingTags, err
	}

	defer errors.DeferredCloser(&err, blobReader)

	var envelopeMetaBuf, bodyBuf bytes.Buffer

	hr := hyphence.Reader{
		Metadata: &envelopeMetaBuf,
		Blob:     &bodyBuf,
	}

	if _, err = hr.ReadFrom(blobReader); err != nil {
		err = errors.Wrap(orgie.ErrBaseUndereferenceable{
			Digest: digestString,
			Cause:  err,
		})
		return base, wasGrouped, groupingTags, err
	}

	envelopeMetadata := orgie.NewMetadata(ids.RepoId{})

	if _, err = envelopeMetadata.ReadFrom(&envelopeMetaBuf); err != nil {
		err = errors.Wrap(orgie.ErrBaseUndereferenceable{
			Digest: digestString,
			Cause:  err,
		})
		return base, wasGrouped, groupingTags, err
	}

	if groupBy, ok := envelopeMetadata.SettingSet.GetByKey("group-by"); ok {
		if ocwk, isKeyed := groupBy.(orgie.SettingWithKey); isKeyed {
			if gb, isGroupBy := ocwk.Setting.(*orgie.SettingGroupBy); isGroupBy &&
				gb.Value != "" {
				wasGrouped = true

				var tags []ids.TagStruct

				for _, value := range strings.Split(gb.Value, ",") {
					var tag ids.TagStruct

					if err = tag.Set(value); err != nil {
						err = errors.Wrap(orgie.ErrBaseUndereferenceable{
							Digest: digestString,
							Cause:  err,
						})
						return base, wasGrouped, groupingTags, err
					}

					tags = append(tags, tag)
				}

				groupingTags = collections_slice.MakeFromSlice(tags...)
			}
		}
	}

	// Mirrors ReadOrganizeFile.Run's exact scaffold
	// (repo_actions/read_organize_file.go) so the base's own document
	// metadata (query tags, -dry-run, etc, excised of nothing -- it's
	// the OUTER document's rendering, one level of nesting in) parses
	// through the identical path patch does.
	baseFlags := orgie.MakeFlags()
	ApplyToOrganizeOptions(repo, &baseFlags.Options)

	baseTextOptions := baseFlags.GetOptionsWithMetadata(
		repo.GetConfig().GetPrintOptions(),
		repo.SkuFormatBoxCheckedOutNoColor(),
		repo.GetStore().GetAbbrStore().GetAbbr(),
		sku.ObjectFactory{},
		orgie.NewMetadata(ids.RepoId{}),
	)

	if base, err = orgie.New(baseTextOptions); err != nil {
		err = errors.Wrap(err)
		return base, wasGrouped, groupingTags, err
	}

	if _, err = base.ReadFrom(&bodyBuf); err != nil {
		err = errors.Wrap(orgie.ErrBaseUndereferenceable{
			Digest: digestString,
			Cause:  err,
		})
		return base, wasGrouped, groupingTags, err
	}

	return base, wasGrouped, groupingTags, err
}
