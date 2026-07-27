package orgie

import (
	"strings"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/0/options_print"
	"code.linenisgreat.com/dodder/go/internal/alfa/string_format_writer"
	"code.linenisgreat.com/dodder/go/internal/bravo/checked_out_state"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/delta/repo_configs"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/golf/box_format"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

// Fixture helpers below deliberately avoid naming *ui.T as a parameter
// type: ui.MakeT's actual return type is an internal package type from a
// different module that this package cannot name directly, so any helper
// declared to accept it by explicit type fails to compile even though
// calling methods on the value works fine. Panic on construction failure
// instead -- these are test-fixture builders, not code under test.

func mustTag(v string) (tag ids.TagStruct) {
	errors.PanicIfError(tag.Set(v))
	return tag
}

// makeThreeWayTestObject constructs a standalone sku.SkuType with the
// given zettel id.
func makeThreeWayTestObject(zettelId string) (object sku.SkuType) {
	object, _ = sku.MakeSkuType() //repool:owned

	var h ids.ZettelId
	errors.PanicIfError(h.Set(zettelId))

	errors.PanicIfError(
		object.GetSkuExternal().GetObjectIdMutable().SetWithId(ids.MustObjectId(h)),
	)

	return object
}

func objectTagStrings(object sku.SkuType) (out []string) {
	for tag := range object.GetSkuExternal().GetMetadata().GetTags().All() {
		out = append(out, tag.String())
	}

	return out
}

func assertTagStringsEqualUnordered(
	t1 *testing.T,
	expected, actual []string,
) {
	t1.Helper()

	if len(expected) != len(actual) {
		t1.Errorf("\nexpected: %q\n  actual: %q", expected, actual)
		return
	}

	seen := make(map[string]bool, len(expected))

	for _, e := range expected {
		seen[e] = true
	}

	for _, a := range actual {
		if !seen[a] {
			t1.Errorf("\nexpected: %q\n  actual: %q", expected, actual)
			return
		}
	}
}

// TestDispatchRemovalGroupedClearsOnlyGroupedDimension pins dodder#374(b)
// plan §6's grouped branch: a grouped document's line-deletion clears ONLY
// the -group-by-matching tag(s) (via ids.IntersectPrefixes, prefix-matching
// the same way Subset does at generation time), leaving every other tag on
// the object untouched -- never the invoking query's selection tags.
func TestDispatchRemovalGroupedClearsOnlyGroupedDimension(t1 *testing.T) {
	t := ui.MakeT(t1)

	object := makeThreeWayTestObject("one/uno")

	objects.SetTags(
		object.GetSkuExternal().GetMetadataMutable(),
		ids.MakeTagSetMutable(
			mustTag("priority-1"),
			mustTag("task"),
		),
	)

	groupingTags := ids.TagSlice{mustTag("priority")}

	unresolved, err := DispatchRemoval(
		RemovalIntent{Object: object},
		true,
		groupingTags,
		NewMetadata(ids.RepoId{}),
	)
	t.AssertNoError(err)

	if unresolved != nil {
		t1.Errorf("expected no unresolved intent, got %#v", unresolved)
	}

	assertTagStringsEqualUnordered(
		t1,
		[]string{"task"},
		objectTagStrings(object),
	)
}

// TestDispatchRemovalUngroupedDelegatesToRemoveFromTransacted pins
// dodder#374(b) plan §6's ungrouped branch: an ungrouped document's
// line-deletion strips the invoking query's selection tag(s) via the
// existing Metadata.RemoveFromTransacted -- today's dodder#374(d)-documented
// behavior, preserved verbatim. The grouped-dimension distinction is
// irrelevant here: RemoveFromTransacted strips whatever tags the
// selectionMetadata carries, regardless of the object's other tags.
func TestDispatchRemovalUngroupedDelegatesToRemoveFromTransacted(t1 *testing.T) {
	t := ui.MakeT(t1)

	object := makeThreeWayTestObject("one/uno")

	objects.SetTags(
		object.GetSkuExternal().GetMetadataMutable(),
		ids.MakeTagSetMutable(
			mustTag("tag-3"),
			mustTag("tag-4"),
			mustTag("new-etikett-for-all"),
		),
	)

	selectionMetadata := NewMetadata(ids.RepoId{})
	selectionMetadata.TagSet = ids.MakeTagSetMutable(
		mustTag("new-etikett-for-all"),
	)

	unresolved, err := DispatchRemoval(
		RemovalIntent{Object: object},
		false,
		nil,
		selectionMetadata,
	)
	t.AssertNoError(err)

	if unresolved != nil {
		t1.Errorf("expected no unresolved intent, got %#v", unresolved)
	}

	assertTagStringsEqualUnordered(
		t1,
		[]string{"tag-3", "tag-4"},
		objectTagStrings(object),
	)
}

func makeThreeWayOptions() Options {
	return Options{
		wasMade:       true,
		Config:        &repo_configs.DryRunOnly{},
		ObjectFactory: (&sku.ObjectFactory{}).SetDefaultsIfNecessary(),
		Metadata:      NewMetadata(ids.RepoId{}),
		Skus:          sku.MakeSkuTypeSetMutable(),
		fmtBox: box_format.MakeBoxCheckedOut(
			string_format_writer.ColorOptions{},
			options_print.Options{},
			nil,
			ids.Abbr{},
			nil,
			nil,
			nil,
		),
	}
}

func mustParseText(body string) *Text {
	text, err := New(makeThreeWayOptions())
	errors.PanicIfError(err)

	_, err = text.ReadFrom(strings.NewReader(body))
	errors.PanicIfError(err)

	return text
}

func makeLiveSet(objectsIn ...sku.SkuType) sku.SkuTypeSet {
	live := sku.MakeSkuTypeSetMutable()

	for _, o := range objectsIn {
		errors.PanicIfError(live.Add(o))
	}

	return live
}

// TestComputeThreeWayDetectsRemoval pins the base/patch removal detection
// (dodder#374(b) plan §5): an object present in base but absent from patch
// entirely (not under ANY heading -- Text.GetSkus resolves by object
// identity across the whole tree, so a move between headings is NOT a
// removal) shows up in ThreeWayResult.Removals.
func TestComputeThreeWayDetectsRemoval(t1 *testing.T) {
	t := ui.MakeT(t1)

	base := mustParseText("- [one/uno tag-a] desc\n- [one/dos tag-a] other\n")
	patch := mustParseText("- [one/dos tag-a] other\n")

	liveUno := makeThreeWayTestObject("one/uno")
	liveDos := makeThreeWayTestObject("one/dos")
	live := makeLiveSet(liveUno, liveDos)

	result, err := ComputeThreeWay(options_print.Options{}, ThreeWayInputs{
		Base:  base,
		Patch: patch,
		Live:  live,
	})
	t.AssertNoError(err)

	if len(result.Removals) != 1 {
		t1.Fatalf("expected exactly 1 removal, got %d: %#v", len(result.Removals), result.Removals)
	}

	removedId := result.Removals[0].Object.GetSkuExternal().GetObjectId().String()
	if removedId != "one/uno" {
		t1.Errorf("expected removed object \"one/uno\", got %q", removedId)
	}
}

// TestComputeThreeWayNoConflictWhenLiveMatchesBase pins the conflict pass's
// negative case: patch touched an object's tags, but live is unchanged
// from base (no drift) -- not a conflict, since only one side moved.
func TestComputeThreeWayNoConflictWhenLiveMatchesBase(t1 *testing.T) {
	t := ui.MakeT(t1)

	base := mustParseText("- [one/uno tag-a] desc\n")
	patch := mustParseText("- [one/uno tag-b] desc\n")

	// live must match base on every patchable aspect (tags AND
	// description AND type), not just tags, so this fixture genuinely
	// represents "live is unchanged from base" post-generalization
	// (2026-07-26 review: drift scope covers all patchable aspects).
	liveUno := makeThreeWayTestObject("one/uno")
	objects.SetTags(
		liveUno.GetSkuExternal().GetMetadataMutable(),
		ids.MakeTagSetMutable(mustTag("tag-a")),
	)
	errors.PanicIfError(
		liveUno.GetSkuExternal().GetMetadataMutable().GetDescriptionMutable().Set("desc"),
	)
	live := makeLiveSet(liveUno)

	result, err := ComputeThreeWay(options_print.Options{}, ThreeWayInputs{
		Base:  base,
		Patch: patch,
		Live:  live,
	})
	t.AssertNoError(err)

	if len(result.Conflicts) != 0 {
		t1.Errorf("expected no conflicts, got %#v", result.Conflicts)
	}
}

// TestComputeThreeWayNoConflictForUntrackedNeverCommittedObject pins a real
// false-positive found via a bats regression (checkin/add -organize on
// brand-new untracked files): base's tags for a not-yet-created object
// (e.g. `- [1.md]`) are organize's PROPOSAL for the new zettel (workspace
// default tags), never actually written to the store -- there is no real
// "previous version" to have drifted from, the same way a brand-new file
// has no git diff base to be "modified" relative to. A live object whose
// state is checked_out_state.Untracked (the query resolved the key to a
// synthesized "this external id exists on disk" placeholder, not a real
// prior commit) must never be treated as drifted, no matter how much its
// (empty) tags differ from base's proposal.
func TestComputeThreeWayNoConflictForUntrackedNeverCommittedObject(t1 *testing.T) {
	t := ui.MakeT(t1)

	base := mustParseText("- [one/uno tag-a tag-b] desc\n")
	patch := mustParseText("- [one/uno tag-a] desc\n")

	liveUno := makeThreeWayTestObject("one/uno")
	errors.PanicIfError(liveUno.SetState(checked_out_state.Untracked))
	live := makeLiveSet(liveUno)

	result, err := ComputeThreeWay(options_print.Options{}, ThreeWayInputs{
		Base:  base,
		Patch: patch,
		Live:  live,
	})
	t.AssertNoError(err)

	if len(result.Conflicts) != 0 {
		t1.Errorf("expected no conflicts for a never-committed live object, got %#v", result.Conflicts)
	}
}

// TestComputeThreeWayConflictWhenBothPatchAndLiveDrift pins the conflict
// pass's positive case (dodder#374(b) plan §4, first-pass object-level
// simplification blessed for v1): patch and live BOTH independently
// changed the same object's tags away from base, to DIFFERENT values --
// this is the drift ComputeThreeWay must flag rather than silently
// picking a side, deferring true per-tag conflict discrimination to the
// mergetool milestone.
func TestComputeThreeWayConflictWhenBothPatchAndLiveDrift(t1 *testing.T) {
	t := ui.MakeT(t1)

	base := mustParseText("- [one/uno tag-a] desc\n")
	patch := mustParseText("- [one/uno tag-b] desc\n")

	// Isolate tag drift specifically -- match base's description so
	// the conflict this test asserts is attributable to tags alone,
	// not incidentally also to a description mismatch.
	liveUno := makeThreeWayTestObject("one/uno")
	objects.SetTags(
		liveUno.GetSkuExternal().GetMetadataMutable(),
		ids.MakeTagSetMutable(mustTag("tag-c")),
	)
	errors.PanicIfError(
		liveUno.GetSkuExternal().GetMetadataMutable().GetDescriptionMutable().Set("desc"),
	)
	live := makeLiveSet(liveUno)

	result, err := ComputeThreeWay(options_print.Options{}, ThreeWayInputs{
		Base:  base,
		Patch: patch,
		Live:  live,
	})
	t.AssertNoError(err)

	if len(result.Conflicts) != 1 {
		t1.Fatalf("expected exactly 1 conflict, got %d: %#v", len(result.Conflicts), result.Conflicts)
	}

	conflictedId := result.Conflicts[0].Object.GetSkuExternal().GetObjectId().String()
	if conflictedId != "one/uno" {
		t1.Errorf("expected conflicted object \"one/uno\", got %q", conflictedId)
	}
}

// TestComputeThreeWayConflictWhenDescriptionDriftsOnTouchedObject mirrors
// TestComputeThreeWayConflictWhenBothPatchAndLiveDrift for description
// instead of tags (2026-07-26 review: drift scope must cover every
// patchable aspect, not just tags -- Changes.Changed clones the whole
// patch sku, description included, so the guard must match). Patch edits
// the description; live independently drifted the description to a
// different value; tags are untouched on both sides -- must conflict.
func TestComputeThreeWayConflictWhenDescriptionDriftsOnTouchedObject(t1 *testing.T) {
	t := ui.MakeT(t1)

	base := mustParseText("- [one/uno tag-a] desc-base\n")
	patch := mustParseText("- [one/uno tag-a] desc-patch\n")

	liveUno := makeThreeWayTestObject("one/uno")
	objects.SetTags(
		liveUno.GetSkuExternal().GetMetadataMutable(),
		ids.MakeTagSetMutable(mustTag("tag-a")),
	)

	if err := liveUno.GetSkuExternal().GetMetadataMutable().GetDescriptionMutable().Set("desc-live"); err != nil {
		t1.Fatalf("failed to set description: %v", err)
	}

	live := makeLiveSet(liveUno)

	result, err := ComputeThreeWay(options_print.Options{}, ThreeWayInputs{
		Base:  base,
		Patch: patch,
		Live:  live,
	})
	t.AssertNoError(err)

	if len(result.Conflicts) != 1 {
		t1.Fatalf("expected exactly 1 conflict, got %d: %#v", len(result.Conflicts), result.Conflicts)
	}

	conflictedId := result.Conflicts[0].Object.GetSkuExternal().GetObjectId().String()
	if conflictedId != "one/uno" {
		t1.Errorf("expected conflicted object \"one/uno\", got %q", conflictedId)
	}
}

// TestComputeThreeWayNoConflictWhenDescriptionDriftsOnUntouchedObject is
// the negative-case sibling: live's description drifted from base, but
// patch never touched this object at all (tags AND description both
// match base) -- the "patch touched it" gate must stay false, so drift
// is silently merged (idempotent), not flagged. Matches the intent
// TestComputeThreeWayNoConflictWhenLiveMatchesBase already pins for
// tags, now covering description.
func TestComputeThreeWayNoConflictWhenDescriptionDriftsOnUntouchedObject(t1 *testing.T) {
	t := ui.MakeT(t1)

	base := mustParseText("- [one/uno tag-a] desc-base\n")
	patch := mustParseText("- [one/uno tag-a] desc-base\n")

	liveUno := makeThreeWayTestObject("one/uno")
	objects.SetTags(
		liveUno.GetSkuExternal().GetMetadataMutable(),
		ids.MakeTagSetMutable(mustTag("tag-a")),
	)

	if err := liveUno.GetSkuExternal().GetMetadataMutable().GetDescriptionMutable().Set("desc-live"); err != nil {
		t1.Fatalf("failed to set description: %v", err)
	}

	live := makeLiveSet(liveUno)

	result, err := ComputeThreeWay(options_print.Options{}, ThreeWayInputs{
		Base:  base,
		Patch: patch,
		Live:  live,
	})
	t.AssertNoError(err)

	if len(result.Conflicts) != 0 {
		t1.Errorf("expected no conflicts (patch never touched this object), got %#v", result.Conflicts)
	}
}

// TestComputeThreeWayConflictWhenTypeDriftsOnTouchedObject is the type
// sibling of the description tests above -- same rationale, cheap to
// add since Type is on the same aspect set patchableAspectsEqual checks.
func TestComputeThreeWayConflictWhenTypeDriftsOnTouchedObject(t1 *testing.T) {
	t := ui.MakeT(t1)

	base := mustParseText("- [one/uno !type-a tag-a] desc\n")
	patch := mustParseText("- [one/uno !type-b tag-a] desc\n")

	liveUno := makeThreeWayTestObject("one/uno")
	objects.SetTags(
		liveUno.GetSkuExternal().GetMetadataMutable(),
		ids.MakeTagSetMutable(mustTag("tag-a")),
	)

	if err := liveUno.GetSkuExternal().GetMetadataMutable().GetDescriptionMutable().Set("desc"); err != nil {
		t1.Fatalf("failed to set description: %v", err)
	}

	if err := liveUno.GetSkuExternal().GetMetadataMutable().GetTypeMutable().SetType("type-c"); err != nil {
		t1.Fatalf("failed to set type: %v", err)
	}

	live := makeLiveSet(liveUno)

	result, err := ComputeThreeWay(options_print.Options{}, ThreeWayInputs{
		Base:  base,
		Patch: patch,
		Live:  live,
	})
	t.AssertNoError(err)

	if len(result.Conflicts) != 1 {
		t1.Fatalf("expected exactly 1 conflict, got %d: %#v", len(result.Conflicts), result.Conflicts)
	}

	conflictedId := result.Conflicts[0].Object.GetSkuExternal().GetObjectId().String()
	if conflictedId != "one/uno" {
		t1.Errorf("expected conflicted object \"one/uno\", got %q", conflictedId)
	}
}

// TestComputeThreeWayFallbackFetchDetectsDriftWhenObjectFallsOutOfLiveQuery
// pins the dodder#374(b) followup: an object touched by both Base and
// Patch, absent from the pre-resolved Live set (as if it fell out of the
// organize session's tag-based query between generation and apply --
// dodder's only selection mechanism is tags, so this is the realistic
// cause, not deletion; dodder has no hard-delete), must still get a
// live-drift comparison via FetchLiveById rather than being silently
// skipped.
func TestComputeThreeWayFallbackFetchDetectsDriftWhenObjectFallsOutOfLiveQuery(t1 *testing.T) {
	t := ui.MakeT(t1)

	base := mustParseText("- [one/uno tag-a] desc-base\n")
	patch := mustParseText("- [one/uno tag-b] desc-base\n")

	live := makeLiveSet() // empty -- object fell out of the live query

	fetched := makeThreeWayTestObject("one/uno")
	objects.SetTags(
		fetched.GetSkuExternal().GetMetadataMutable(),
		ids.MakeTagSetMutable(mustTag("tag-c")), // drifted from base's tag-a
	)
	errors.PanicIfError(
		fetched.GetSkuExternal().GetMetadataMutable().GetDescriptionMutable().Set("desc-base"),
	)

	var fetchCalled bool

	result, err := ComputeThreeWay(options_print.Options{}, ThreeWayInputs{
		Base:  base,
		Patch: patch,
		Live:  live,
		FetchLiveById: func(objectId *ids.ObjectId) (sku.SkuType, bool, error) {
			fetchCalled = true
			return fetched, true, nil
		},
	})
	t.AssertNoError(err)

	if !fetchCalled {
		t1.Fatalf("expected FetchLiveById to be called")
	}

	if len(result.Conflicts) != 1 {
		t1.Fatalf("expected exactly 1 conflict, got %d: %#v", len(result.Conflicts), result.Conflicts)
	}

	conflictedId := result.Conflicts[0].Object.GetSkuExternal().GetObjectId().String()
	if conflictedId != "one/uno" {
		t1.Errorf("expected conflicted object \"one/uno\", got %q", conflictedId)
	}
}

// TestComputeThreeWayFallbackFetchNoConflictWhenLiveMatchesBase is the
// negative sibling: FetchLiveById returns an object matching base on
// every patchable aspect, so no drift is detected -- the fallback must
// not over-trigger just because it was consulted.
func TestComputeThreeWayFallbackFetchNoConflictWhenLiveMatchesBase(t1 *testing.T) {
	t := ui.MakeT(t1)

	base := mustParseText("- [one/uno tag-a] desc-base\n")
	patch := mustParseText("- [one/uno tag-b] desc-base\n")

	live := makeLiveSet() // empty -- object fell out of the live query

	fetched := makeThreeWayTestObject("one/uno")
	objects.SetTags(
		fetched.GetSkuExternal().GetMetadataMutable(),
		ids.MakeTagSetMutable(mustTag("tag-a")), // matches base
	)
	errors.PanicIfError(
		fetched.GetSkuExternal().GetMetadataMutable().GetDescriptionMutable().Set("desc-base"),
	)

	result, err := ComputeThreeWay(options_print.Options{}, ThreeWayInputs{
		Base:  base,
		Patch: patch,
		Live:  live,
		FetchLiveById: func(objectId *ids.ObjectId) (sku.SkuType, bool, error) {
			return fetched, true, nil
		},
	})
	t.AssertNoError(err)

	if len(result.Conflicts) != 0 {
		t1.Errorf("expected no conflicts (live matches base), got %#v", result.Conflicts)
	}
}

// TestComputeThreeWayFallbackFetchSkipsWhenGenuinelyNeverCommitted covers
// FetchLiveById's other outcome: (nil, false, nil), meaning the object
// really was never committed (e.g. still a pending new-object proposal).
// Must behave exactly like the no-fallback-configured case: skip, no
// conflict, no error.
func TestComputeThreeWayFallbackFetchSkipsWhenGenuinelyNeverCommitted(t1 *testing.T) {
	t := ui.MakeT(t1)

	base := mustParseText("- [one/uno tag-a] desc-base\n")
	patch := mustParseText("- [one/uno tag-b] desc-base\n")

	live := makeLiveSet() // empty

	result, err := ComputeThreeWay(options_print.Options{}, ThreeWayInputs{
		Base:  base,
		Patch: patch,
		Live:  live,
		FetchLiveById: func(objectId *ids.ObjectId) (sku.SkuType, bool, error) {
			return nil, false, nil
		},
	})
	t.AssertNoError(err)

	if len(result.Conflicts) != 0 {
		t1.Errorf("expected no conflicts (never committed), got %#v", result.Conflicts)
	}
}
