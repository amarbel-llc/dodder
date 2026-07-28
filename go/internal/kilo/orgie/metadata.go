package orgie

import (
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/0/hyphence"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/ohio"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter_set"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/format"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/pool"
)

type TagSetGetter interface {
	GetTags() ids.TagSet
}

func NewMetadata(repoId ids.RepoId) Metadata {
	return Metadata{
		RepoId:     repoId,
		TagSet:     ids.MakeTagSetFromSlice(),
		SettingSet: MakeSettingSet(nil),
	}
}

func NewMetadataWithSettingLookup(
	repoId ids.RepoId,
	elements map[string]Setting,
) Metadata {
	return Metadata{
		RepoId:     repoId,
		TagSet:     ids.MakeTagSetFromSlice(),
		SettingSet: MakeSettingSet(elements),
	}
}

// TODO replace with embedded *sku.Transacted
type Metadata struct {
	ids.TagSet
	Matchers interfaces.Set[sku.Query] // TODO remove
	SettingSet
	Type   ids.TypeStruct
	RepoId ids.RepoId
}

func (metadata *Metadata) GetTags() ids.TagSet {
	return metadata.TagSet
}

func (metadata *Metadata) SetFromObjectMetadata(
	otherMetadata objects.MetadataMutable,
	repoId ids.RepoId,
) (err error) {
	metadata.TagSet = ids.CloneTagSet(otherMetadata.GetTags())

	for comment := range otherMetadata.GetIndex().GetComments() {
		if err = metadata.SettingSet.Set(comment); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	metadata.Type = otherMetadata.GetType().ToType()

	return err
}

func (metadata Metadata) RemoveFromTransacted(object sku.SkuType) (err error) {
	tags := ids.CloneTagSetMutable(object.GetSkuExternal().GetMetadata().GetTags())

	for element := range metadata.All() {
		quiter_set.Del(tags, element)
	}

	objects.SetTags(object.GetSkuExternal().GetMetadataMutable(), tags)

	return err
}

func (metadata Metadata) AsMetadata() (m1 objects.MetadataMutable) {
	m1 = objects.Make()
	m1.GetTypeMutable().ResetWithObjectId(metadata.Type)
	objects.SetTags(m1, metadata.TagSet)
	return m1
}

func (metadata Metadata) GetMetadataWriterTo() hyphence.MetadataWriterTo {
	return metadata
}

func (metadata Metadata) HasMetadataContent() bool {
	if metadata.Len() > 0 {
		return true
	}

	if !metadata.Type.IsEmpty() {
		return true
	}

	if len(metadata.SettingSet.Settings) > 0 {
		return true
	}

	return false
}

func (metadata *Metadata) ReadFrom(reader io.Reader) (n int64, err error) {
	bufferedReader, repool := pool.GetBufferedReader(reader)
	defer repool()

	tagSet := ids.MakeTagSetMutable()
	addTag := quiter.MakeFuncAddString(tagSet)

	// `_`-reserved settings fields (dodder#374, cutting-garden RFC 0015):
	// `- _key = value` is the settings-field spelling (spaced `=` is
	// normative per RFC 0015's merged two-plane revision -- ruled
	// 2026-07-28 to apply uniformly, no exemption for already-shipped
	// `_base`/`_group-by`). `% key:value` (the SettingSet path above)
	// remains accepted as a deprecated alias during migration.
	//
	// TrimSpace on both the key and value halves accepts the spaced form
	// unconditionally and the pre-RFC-0015 unspaced form for free (a
	// no-op trim when there's no whitespace to remove) -- no separate
	// legacy-vs-new branch needed for spacing specifically, unlike the
	// colon-vs-equals `%` legacy alias above.
	//
	// A key this build/context has no prototype for at all (e.g.
	// `_dry-run = true` read without `-dry-run` on the CLI, so the
	// "dry-run" prototype was never registered) is a legitimate,
	// expected no-op -- exactly how an unregistered `%` comment has
	// always behaved (SettingSet.Set falls back to
	// SettingUnknown). That parity is intentional, not a gap: the
	// prototype registry is inherently context-dependent, so a settings
	// field this run doesn't recognize can't be distinguished from one
	// it simply isn't active for.
	//
	// A key that IS registered but explicitly is not a settings field
	// (e.g. the built-in "hide", whose Set() is an unimplemented stub)
	// is different: routing it through SettingSet.Set would hit
	// that stub and surface an opaque, unrelated error. Reject it here
	// instead by falling through to addTag(v), which gives the same
	// clear "not a valid tag" diagnostic any other `=`-containing `-`
	// line gets.
	addTagOrSettingsField := func(v string) (err error) {
		if key, value, ok := strings.Cut(v, "="); ok && strings.HasPrefix(key, "_") {
			prototypeKey := strings.TrimSpace(strings.TrimPrefix(key, "_"))
			value = strings.TrimSpace(value)
			proto, registered := metadata.SettingSet.GetPrototypeSettings()[prototypeKey]

			if !registered || isSettingsField(proto) {
				if err = metadata.SettingSet.Set(prototypeKey + ":" + value); err != nil {
					err = errors.Wrap(err)
					return err
				}

				return err
			}
		}

		return addTag(v)
	}

	dataPlaneReader := ohio.MakeLineReaderKeyValues(
		map[string]interfaces.FuncSetString{
			"-": addTagOrSettingsField,
			"!": metadata.Type.Set,
		},
	)

	// ohio.MakeLineReaderKeyValues dispatches by looking up the exact
	// text before a line's FIRST SPACE as a map key -- not a prefix
	// match on "%". A `%:<name> = value` directive's first space falls
	// AFTER the name (before "="), so ohio's lookup can never represent
	// a `%:` family for any directive name, ever (confirmed against
	// ohio/line_reader.go). Short-circuiting on the raw "%" prefix
	// BEFORE ohio sees the line -- rather than changing ohio itself,
	// a general-purpose lib/alfa primitive with other consumers -- keeps
	// this entirely dodder-local. "-"/"!" lines are unaffected: same
	// ohio call, just with "%" dropped from its map.
	combined := func(line string) (err error) {
		if strings.HasPrefix(line, "%") {
			return metadata.readOperationalPlaneLine(line)
		}

		return dataPlaneReader(line)
	}

	if n, err = format.ReadLines(
		bufferedReader,
		ohio.MakeLineReaderRepeat(combined),
	); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	metadata.TagSet = ids.CloneTagSet(tagSet)

	return n, err
}

// readOperationalPlaneLine dispatches one raw "%"-prefixed line (leading
// "%" INCLUDED) -- cutting-garden RFC 0015 (merged, ruled 2026-07-28):
// the character adjacent to "%" is the ENTIRE distinction between the
// operational plane's two shapes:
//
//   - ":" (colon, no space) -- a semantic directive, routed through
//     SetDirective against the prototype registry (SettingSet.Set's
//     colon-splitting logic doesn't apply here at all; a directive's
//     own separator is "=", not ":" -- the colon was already consumed
//     as the "%:" sigil).
//   - " " (space) -- inert prose, never parsed for structure, recorded
//     via AddInertProse for round-trip fidelity only.
//
// `% dry-run:<value>` is the ONE pre-RFC-0015 legacy comment alias kept
// for back-compat -- dry-run had no settings-field spelling before
// dodder#374(c) existed, so it's a narrow, named exception, not a
// general "any % line might be key:value" rule (which is what the
// pre-RFC-0015 SettingSet.Set path did for every "%" line, and which
// silently discarded genuine prose containing a colon, e.g.
// "% meeting at 3:00" -- a latent bug this design doesn't repeat).
func (metadata *Metadata) readOperationalPlaneLine(line string) (err error) {
	rest := strings.TrimPrefix(line, "%")

	if strings.HasPrefix(rest, ":") {
		if err = metadata.SettingSet.SetDirective(rest[1:]); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}

	prose := strings.TrimPrefix(rest, " ")

	if legacyKey, legacyValue, ok := strings.Cut(prose, ":"); ok && legacyKey == "dry-run" {
		if err = metadata.SettingSet.setDirectiveLegacyAlias("dry-run = " + legacyValue); err != nil {
			err = errors.Wrap(err)
			return err
		}

		return err
	}

	metadata.SettingSet.AddInertProse(prose)

	return err
}

func (metadata Metadata) WriteTo(w1 io.Writer) (n int64, err error) {
	return metadata.writeTo(w1, true)
}

// WriteDataPlaneTo renders only the DATA plane (`-`/`!` lines) --
// cutting-garden RFC 0015 (merged): the organize-base-v1 base blob's
// digest must depend only on the document's data plane, never the
// operational plane (`%`/`%:`), so a document generated with or without
// operational directives produces the same base. Used exclusively by
// base-blob generation (repo_actions.WriteOrganizeBaseAndActivate);
// normal document rendering (what the user is shown/edits) stays on
// WriteTo, which keeps interleaving both planes as before.
func (metadata Metadata) WriteDataPlaneTo(w1 io.Writer) (n int64, err error) {
	return metadata.writeTo(w1, false)
}

func (metadata Metadata) writeTo(w1 io.Writer, includeOperationalPlane bool) (n int64, err error) {
	w := format.NewLineWriter()

	for _, o := range metadata.SettingSet.Settings {
		// `_`-reserved settings fields (dodder#374, cutting-garden RFC
		// 0015): a Setting implementing SettingAsField is
		// written as `- _key=value`, not a `%` comment, so it isn't opaque
		// per hyphence RFC 0001. Everything else keeps the comment spelling.
		// Migrating a setting is a one-line change on that Setting
		// (implement IsSettingsField) -- this loop needs no new branch.
		if isSettingsField(o) {
			ocwk := o.(SettingWithKey)
			// Spaced "=" per RFC 0015's merged two-plane revision
			// (ruled 2026-07-28: normative for ALL metadata lines,
			// no exemption for _base/_group-by).
			w.WriteFormat("- _%s = %s", ocwk.Key, ocwk.Setting)
			continue
		}

		if !includeOperationalPlane {
			continue
		}

		// Operational-plane items split into two write-side shapes by
		// the per-INSTANCE IsDirective marker (SettingWithKey's doc
		// comment explains why this can't be a per-TYPE distinction):
		// IsDirective -- came through setDirective (RFC 0015's `%:name
		// = value` syntax or its `% dry-run:true` legacy alias) -- is a
		// "%:key = value" directive; everything else (AddInertProse's
		// unwrapped prose, OR a not-yet-migrated legacy keyed comment
		// like checkin's "delete" flag, still constructed via the old
		// AddPrototypeAndOption path until piece 4) keeps the
		// pre-RFC-0015 "% text" / "% key:value" spelling unchanged.
		if ocwk, isKeyed := o.(SettingWithKey); isKeyed && ocwk.IsDirective {
			w.WriteFormat("%%:%s = %s", ocwk.Key, ocwk.Setting)
			continue
		}

		w.WriteFormat("%% %s", o)
	}

	for _, e := range quiter.SortedStrings(metadata.TagSet) {
		w.WriteFormat("- %s", e)
	}

	tString := metadata.Type.StringSansOp()

	if tString != "" {
		w.WriteFormat("! %s", tString)
	}

	if includeOperationalPlane && metadata.Matchers != nil {
		for _, c := range quiter.SortedStrings(metadata.Matchers) {
			w.WriteFormat("%% Matcher:%s", c)
		}
	}

	return w.WriteTo(w1)
}

// dataPlaneOnlyMetadata adapts Metadata.WriteDataPlaneTo to
// hyphence.MetadataWriterTo (io.WriterTo + HasMetadataContent) so
// Text.WriteDataPlaneTo (main.go) can swap it in for hyphence.Writer's
// Metadata field without duplicating Text.WriteTo's body. HasMetadataContent
// is promoted from the embedded Metadata unchanged -- plane filtering only
// affects WHICH lines render, not whether there's content to report.
type dataPlaneOnlyMetadata struct {
	Metadata
}

func (m dataPlaneOnlyMetadata) WriteTo(w io.Writer) (int64, error) {
	return m.Metadata.WriteDataPlaneTo(w)
}
