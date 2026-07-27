package orgie

import (
	"fmt"
	"sort"

	"code.linenisgreat.com/dodder/go/internal/0/options_print"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

func MakeSkuMapWithOrder(c int) (out SkuMapWithOrder) {
	out.m = make(map[string]skuTypeWithIndex, c)
	return out
}

type skuTypeWithIndex struct {
	sku sku.SkuType
	int
}

type SkuMapWithOrder struct {
	m    map[string]skuTypeWithIndex
	next int
}

func (smwo *SkuMapWithOrder) AsExternalLikeSet() sku.SkuTypeSetMutable {
	elms := sku.MakeSkuTypeSetMutable()

	for _, sk := range smwo.AllSkuAndIndex() {
		errors.PanicIfError(elms.Add(sk))
	}

	return elms
}

func (smwo *SkuMapWithOrder) AsTransactedSet() sku.TransactedMutableSet {
	tms := sku.MakeTransactedMutableSet()

	for _, el := range smwo.AllSkuAndIndex() {
		errors.PanicIfError(tms.Add(el.GetSkuExternal()))
	}

	return tms
}

func (sm *SkuMapWithOrder) Del(sk sku.SkuType) error {
	delete(sm.m, keyer.GetKey(sk))
	return nil
}

func (sm *SkuMapWithOrder) Add(sk sku.SkuType) error {
	k := keyer.GetKey(sk)
	entry, ok := sm.m[k]

	if !ok {
		entry.int = sm.next
		entry.sku = sk
		sm.next++
	}

	sm.m[k] = entry

	return nil
}

func (sm *SkuMapWithOrder) Len() int {
	return len(sm.m)
}

func (sm *SkuMapWithOrder) Clone() (out SkuMapWithOrder) {
	out = MakeSkuMapWithOrder(sm.Len())

	for _, v := range sm.m {
		out.Add(v.sku)
	}

	return out
}

func (sm SkuMapWithOrder) Sorted() (out []sku.SkuType) {
	out = make([]sku.SkuType, 0, sm.Len())

	for _, v := range sm.m {
		out = append(out, v.sku)
	}

	sort.Slice(out, func(i, j int) bool {
		iObject := out[i].GetSkuExternal()
		jObject := out[j].GetSkuExternal()

		switch {
		case iObject.GetObjectId().IsEmpty() && jObject.GetObjectId().IsEmpty():
			return iObject.GetMetadata().GetDescription().String() < jObject.GetMetadata().GetDescription().String()

		case iObject.GetObjectId().IsEmpty():
			return true

		case jObject.GetObjectId().IsEmpty():
			return false

		default:
			return iObject.GetObjectId().String() < jObject.GetObjectId().String()
		}
	})

	return out
}

func (smwo *SkuMapWithOrder) AllSkuAndIndex() interfaces.Seq2[int, sku.SkuType] {
	return func(yield func(int, sku.SkuType) bool) {
		for i, sk := range smwo.Sorted() {
			if !yield(i, sk) {
				break
			}
		}
	}
}

type Changes struct {
	Before, After  SkuMapWithOrder
	Added, Removed SkuMapWithOrder
	Changed        SkuMapWithOrder
}

func (c Changes) String() string {
	return fmt.Sprintf(
		"Before: %d, After: %d, Added: %d, Removed: %d, Changed: %d",
		c.Before.Len(),
		c.After.Len(),
		c.Added.Len(),
		c.Removed.Len(),
		c.Changed.Len(),
	)
}

// OrganizeResults is the dodder#374(b) three-way apply seam. Before/Original
// keep their pre-(b) field names for call-site compatibility, but as of the
// _base-pinning cutover they carry different values than they used to:
//
//   - Before is now the DEREFERENCED BASE (repo_actions.DereferenceOrganizeBase's
//     return value) -- what `organize` generated and the user was shown --
//     not a freshly-regenerated organize file at commit time.
//   - Original is now LIVE -- the store's current state, read at APPLY time
//     (repo_actions is responsible for re-reading it fresh, not reusing the
//     set collected at generation time), so drift between generation and
//     apply is detectable (RFC 0015; see three_way.go's ComputeThreeWay).
//   - WasGrouped/GroupingTags are DereferenceOrganizeBase's other two return
//     values, threaded through so ChangesFromResults can dispatch removals
//     correctly (three_way.go's DispatchRemoval).
//
// This new meaning only holds for callers that route through
// repo_actions.PrepareOrganizeResultsForApply (LockAndCommitOrganizeResults,
// checkin.go's runOrganize). The legacy repo_actions.Organize struct
// (checkout.go/clean.go's `-organize` flag, organize_remote.go's pre-pull
// narrowing) still populates Before/Original directly from the same
// generation-time query snapshot, with no `_base` dereference or live
// requery -- for those callers ComputeThreeWay's Base and Live are the same
// snapshot by construction, so conflict detection is a structural no-op
// (never fires, same as pre-(b)); this is a known, deliberately unaddressed
// gap in this feature's scope, not a regression (reviewed and confirmed
// unnecessary for those three callers: all read-only filtering or
// idempotent checked-out-state deletion, never a tag/description commit).
//
// FetchLiveById (optional) is PrepareOrganizeResultsForApply's ID-based
// fallback for ComputeThreeWay's live-drift check, threaded through
// unchanged -- see ThreeWayInputs' own doc comment (three_way.go) for why
// this exists and why it's a plain closure rather than repo/store access
// gained here. Only PrepareOrganizeResultsForApply-routed callers set it;
// the legacy repo_actions.Organize path above leaves it nil.
type OrganizeResults struct {
	Before, After *Text
	Original      sku.SkuTypeSet
	QueryGroup    *queries.Query
	WasGrouped    bool
	GroupingTags  ids.TagSlice
	FetchLiveById func(objectId *ids.ObjectId) (sku.SkuType, bool, error)
}

func ChangesFrom(
	po options_print.Options,
	a, b *Text,
	original sku.SkuTypeSet,
) (c Changes, err error) {
	if c, err = ChangesFromResults(
		po,
		OrganizeResults{
			Before:   a,
			After:    b,
			Original: original,
		},
	); err != nil {
		err = errors.Wrap(err)
		return c, err
	}

	return c, err
}

// ChangesFromResults is dodder#374(b) plan §4's "thin wrapper": it excises
// `_base` from the patch (structural, matching what generation never wrote
// in the first place -- see ExciseBaseField's sibling logic), then delegates
// to ComputeThreeWay (three_way.go) for the base/patch/live diff, then
// dispatches each removal (DispatchRemoval) to either a grouped-dimension
// tag-clear or the ungrouped RemoveFromTransacted -- preserving the
// pre-(b) behavior of adding removed-and-mutated objects back into Changed
// so they're included in the commit plan.
//
// Kept kilo-tier (no repo access): base/live are supplied ALREADY resolved
// by the sierra-tier caller (repo_actions), per OrganizeResults' doc comment
// above -- this function does not itself dereference `_base` or query the
// store.
func ChangesFromResults(
	po options_print.Options,
	results OrganizeResults,
) (c Changes, err error) {
	// `_base` is expected to always be present on results.After here --
	// generation always writes it (organize_base.go) and callers are
	// required to reject an apply with it missing before reaching this
	// function -- so the found bool is intentionally discarded.
	results.After.Metadata.OptionCommentSet.RemoveByKey("base")

	var threeWayResult ThreeWayResult

	if threeWayResult, err = ComputeThreeWay(po, ThreeWayInputs{
		Base:          results.Before,
		Patch:         results.After,
		Live:          results.Original,
		WasGrouped:    results.WasGrouped,
		GroupingTags:  results.GroupingTags,
		FetchLiveById: results.FetchLiveById,
	}); err != nil {
		err = errors.Wrap(err)
		return c, err
	}

	if len(threeWayResult.Conflicts) > 0 {
		objectIds := make([]string, len(threeWayResult.Conflicts))

		for i, conflict := range threeWayResult.Conflicts {
			objectIds[i] = conflict.Object.GetSkuExternal().GetObjectId().String()
		}

		err = errors.Wrap(ErrConflicts{ObjectIds: objectIds})

		return c, err
	}

	c = threeWayResult.Changes

	for _, removal := range threeWayResult.Removals {
		var unresolved *UnresolvedIntent

		if unresolved, err = DispatchRemoval(
			removal,
			results.WasGrouped,
			results.GroupingTags,
			results.Before.Metadata,
		); err != nil {
			err = errors.Wrap(err)
			return c, err
		}

		if unresolved != nil {
			// dodder#374(b) plan §6 scaffolding: dodder's only selection
			// mechanism today is tags, which RemoveFromTransacted always
			// resolves, so this is unreachable until a non-tag selection
			// term exists. When it does, surface unresolved to the
			// caller instead of silently dropping it.
			continue
		}

		if err = c.Changed.Add(removal.Object); err != nil {
			err = errors.Wrap(err)
			return c, err
		}
	}

	return c, err
}

func applyToText(
	po options_print.Options,
	t *Text,
) (err error) {
	if po.BoxPrintTagsAlways {
		return err
	}

	for el := range t.Options.Skus.All() {
		sk := el.GetSkuExternal()

		if sk.GetMetadata().GetDescription().IsEmpty() {
			continue
		}

		sk.GetMetadataMutable().ResetTags()
	}

	return err
}
