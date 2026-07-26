package orgie

import (
	"errors"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/0/options_print"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
)

// TestChangesFromResultsSurfacesConflictAsError pins a real bug found
// while writing dodder#374(b)'s test suite: ComputeThreeWay computed
// ThreeWayResult.Conflicts correctly, but ChangesFromResults discarded it
// -- `c = threeWayResult.Changes` silently dropped the conflict set
// entirely, so a genuine base/live drift conflict never reached the
// caller. The plan's §4 says v1 "rejects loudly (mergetool deferred)";
// silently committing over a conflict is the opposite of that. This pins
// ChangesFromResults returning ErrConflicts instead.
func TestChangesFromResultsSurfacesConflictAsError(t1 *testing.T) {
	base := mustParseText("- [one/uno tag-a] desc\n")
	patch := mustParseText("- [one/uno tag-b] desc\n")

	liveUno := makeThreeWayTestObject("one/uno")
	objects.SetTags(
		liveUno.GetSkuExternal().GetMetadataMutable(),
		ids.MakeTagSetMutable(mustTag("tag-c")),
	)
	live := makeLiveSet(liveUno)

	_, err := ChangesFromResults(options_print.Options{}, OrganizeResults{
		Before:   base,
		After:    patch,
		Original: live,
	})

	if err == nil {
		t1.Fatalf("expected ErrConflicts, got nil")
	}

	var conflictErr ErrConflicts
	if !errors.As(err, &conflictErr) {
		t1.Fatalf("expected err to unwrap to ErrConflicts, got: %v", err)
	}

	if len(conflictErr.ObjectIds) != 1 || conflictErr.ObjectIds[0] != "one/uno" {
		t1.Errorf("expected ObjectIds [\"one/uno\"], got: %#v", conflictErr.ObjectIds)
	}
}

// TestChangesFromResultsNoErrorWhenNoConflict is the negative-case
// sibling: an ordinary patch-only change (no live drift) commits
// normally, without ErrConflicts.
func TestChangesFromResultsNoErrorWhenNoConflict(t1 *testing.T) {
	base := mustParseText("- [one/uno tag-a] desc\n")
	patch := mustParseText("- [one/uno tag-b] desc\n")

	liveUno := makeThreeWayTestObject("one/uno")
	objects.SetTags(
		liveUno.GetSkuExternal().GetMetadataMutable(),
		ids.MakeTagSetMutable(mustTag("tag-a")),
	)
	live := makeLiveSet(liveUno)

	_, err := ChangesFromResults(options_print.Options{}, OrganizeResults{
		Before:   base,
		After:    patch,
		Original: live,
	})
	if err != nil {
		t1.Errorf("expected no error, got: %v", err)
	}
}
