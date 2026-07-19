package orgie

import (
	"code.linenisgreat.com/dodder/go/internal/0/options_print"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter_set"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

// ThreeWayInputs holds the three inputs the dodder#374(b) diff engine
// operates on (plan §4/§5), replacing the old two-input Before/After
// diff: Base (dereferenced from `_base`, what the user was shown),
// Patch (the edited document), and Live (a FRESH query against the
// store at apply time -- NOT the query-time snapshot the old
// OrganizeResults.Original carried, since that goes stale the moment
// anything else touches the store between generation and apply; RFC
// 0015 treats base/live as distinct precisely so drift is detectable).
//
// WasGrouped/GroupingTags are threaded from DereferenceOrganizeBase's
// return values (repo_actions/organize_base.go) -- base-authoritative
// by construction, since nothing in this file re-derives them from
// Patch.
type ThreeWayInputs struct {
	Base         *Text
	Patch        *Text
	Live         sku.SkuTypeSet
	WasGrouped   bool
	GroupingTags ids.TagSlice
}

// RemovalIntent is one object present in Base, absent from Patch --
// §5 (this file) only notices the structural fact; what the removal
// MEANS (a tag-clear or an unresolved intent) is DispatchRemoval's job
// (§6), dispatched on WasGrouped.
type RemovalIntent struct {
	Object sku.SkuType
}

// UnresolvedIntent is a first-class apply outcome (RFC 0015's
// "Deletion semantics by grouped-ness"): a removal's write cannot be
// resolved automatically. v1 reports these as structured rejections
// rather than silently dropping or flatly erroring.
//
// Kind is the typed identity of the PROBLEM (e.g.
// "start-date-removal-underdetermined"), NOT the resolution choice --
// per the 2026-07-19 review correction: RFC 0015's batchability ("one
// prompt resolving the same question across N objects") groups by the
// QUESTION, so N objects sharing a Kind batch together. Options is
// rendered FROM Kind at the output boundary (cutting-garden#147
// forward-compat: Kind stays additive if it later becomes a typed
// identifier rather than a bare string).
type UnresolvedIntent struct {
	Object  sku.SkuType
	Term    string
	Kind    string
	Options []string
}

// ConflictedObject is drift (live differs from base) on an object
// Patch ALSO touches -- v1 rejects loudly (mergetool deferred, plan
// §4). First-pass simplification, flagged for review: this is
// object-level (does ANY tag differ), not RFC 0015's field/tag-level
// "same aspect touched" diffing -- an object where patch and live both
// changed DIFFERENT tags is conservatively treated as a conflict too,
// erring toward loud rejection over silently picking a side. Full
// per-tag conflict discrimination is deferred to the mergetool
// milestone (plan's Out-of-scope / Deferred sections).
type ConflictedObject struct {
	Object sku.SkuType
}

type ThreeWayResult struct {
	Changes   Changes
	Removals  []RemovalIntent
	Conflicts []ConflictedObject
}

// ComputeThreeWay is the dodder#374(b) plan §5 engine: patch - base =
// structural intent (moves, edits, creations -- reusing Text.GetSkus,
// the same machinery the old two-input diff used, just fed Base
// instead of a freshly-regenerated "Before" and Live instead of the
// stale query-time "Original"); live - base = drift (new, no existing
// precedent). Objects in Base absent from Patch are collected as
// RemovalIntent, not resolved here -- DispatchRemoval (§6) does that,
// per object, against WasGrouped/GroupingTags.
func ComputeThreeWay(
	po options_print.Options,
	inputs ThreeWayInputs,
) (result ThreeWayResult, err error) {
	if err = applyToText(po, inputs.Base); err != nil {
		err = errors.Wrap(err)
		return result, err
	}

	var baseSkus, patchSkus SkuMapWithOrder

	if baseSkus, err = inputs.Base.GetSkus(inputs.Live); err != nil {
		err = errors.Wrap(err)
		return result, err
	}

	if patchSkus, err = inputs.Patch.GetSkus(inputs.Live); err != nil {
		err = errors.Wrap(err)
		return result, err
	}

	result.Changes.Before = baseSkus
	result.Changes.After = patchSkus
	result.Changes.Changed = patchSkus.Clone()
	result.Changes.Removed = baseSkus.Clone()

	for _, entry := range patchSkus.m {
		if err = result.Changes.Removed.Del(entry.sku); err != nil {
			err = errors.Wrap(err)
			return result, err
		}
	}

	for _, sk := range result.Changes.Removed.AllSkuAndIndex() {
		result.Removals = append(result.Removals, RemovalIntent{Object: sk})
	}

	// Drift: live vs base, for objects both Patch and Base agree exist.
	for _, patchEntry := range patchSkus.m {
		key := keyer.GetKey(patchEntry.sku)

		baseEntry, inBase := baseSkus.m[key]
		if !inBase {
			continue // creation/adoption, not drift
		}

		liveObject, inLive := inputs.Live.Get(key)
		if !inLive {
			continue // gone from the store entirely -- a different failure mode than drift
		}

		patchTouched := !quiter_set.Equals(
			patchEntry.sku.GetSkuExternal().GetMetadata().GetTags(),
			baseEntry.sku.GetSkuExternal().GetMetadata().GetTags(),
		)

		liveDrifted := !quiter_set.Equals(
			liveObject.GetSkuExternal().GetMetadata().GetTags(),
			baseEntry.sku.GetSkuExternal().GetMetadata().GetTags(),
		)

		if patchTouched && liveDrifted {
			result.Conflicts = append(
				result.Conflicts,
				ConflictedObject{Object: patchEntry.sku},
			)
		}
	}

	return result, err
}

// DispatchRemoval implements RFC 0015's "Deletion semantics by
// grouped-ness" (revised 2026-07-18) on one RemovalIntent (plan §6):
//
//   - grouped: clear only the grouped-dimension tag(s) this object had
//     (GroupingTags-matching, via ids.IntersectPrefixes -- the same
//     prefix-matching primitive Subset uses at generation time,
//     set_prefix_transacted.go:170-200). Selection predicates (the
//     invoking query's tags) are NEVER written here.
//   - ungrouped: evaluate the invoking query's selection through
//     mapping writability. dodder's ONLY selection mechanism today is
//     tags (write:many) -- delegates to Metadata.RemoveFromTransacted,
//     today's confirmed, preserved-verbatim behavior
//     (dodder#374(d)-documented). dodder has no field-predicate
//     selection terms reaching this code path today (organize's
//     document metadata TagSet is always tag-derived,
//     queries.GetTags(qg)), so the non-writable/UnresolvedIntent branch
//     is scaffolded per plan §6's own acknowledgment ("likely
//     low-traffic in dodder specifically") but not exercised by
//     anything that exists yet -- always returns unresolved == nil.
func DispatchRemoval(
	removal RemovalIntent,
	wasGrouped bool,
	groupingTags ids.TagSlice,
	selectionMetadata Metadata,
) (unresolved *UnresolvedIntent, err error) {
	if !wasGrouped {
		if err = selectionMetadata.RemoveFromTransacted(removal.Object); err != nil {
			err = errors.Wrap(err)
			return unresolved, err
		}

		return unresolved, err
	}

	objectTags := removal.Object.GetSkuExternal().GetMetadata().GetTags()
	remaining := ids.CloneTagSetMutable(objectTags)

	for _, groupTag := range groupingTags {
		matching := ids.IntersectPrefixes(objectTags, groupTag)

		for tag := range matching.All() {
			remaining.DelKey(tag.String())
		}
	}

	objects.SetTags(removal.Object.GetSkuExternal().GetMetadataMutable(), remaining)

	return unresolved, err
}
