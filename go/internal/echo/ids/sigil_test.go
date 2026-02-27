package ids

import (
	"bytes"
	"testing"

	"code.linenisgreat.com/dodder/go/lib/bravo/ui"
)

func TestSigilContains(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := SigilAll

	if !sut.ContainsOneOf(SigilLatest) {
		t.Errorf("expected SigilAll to contain SigilLatest")
	}

	sut = SigilLatest
	sut.Add(SigilHidden)

	if !sut.ContainsOneOf(SigilLatest) {
		t.Errorf("expected sut to contain SigilTail")
	}

	if !sut.ContainsOneOf(SigilHidden) {
		t.Errorf("expected sut to contain SigilHidden")
	}

	if !sut.ContainsOneOf(sut) {
		t.Errorf("expected sut to contain sut")
	}

	if !sut.Contains(sut) {
		t.Errorf("expected sut to contain sut")
	}

	other := SigilHistory
	other.Add(SigilHidden)

	if !sut.ContainsOneOf(other) {
		t.Errorf("expected sut to contain one ofother")
	}

	if sut.Contains(other) {
		t.Errorf("expected sut not to contain other")
	}

	if sut.ContainsOneOf(SigilExternal) {
		t.Errorf("expected sut not to contain SigilCwd")
	}
}

func TestSigilReadWrite(t1 *testing.T) {
	t := ui.T{T: t1}

	sut := SigilAll
	b := bytes.NewBuffer(nil)

	{
		n, err := sut.WriteTo(b)
		t.AssertNoError(err)
		if n != 1 {
			t.PrintDiff(1, n)
		}
	}

	var actual Sigil

	{
		n, err := actual.ReadFrom(b)
		t.AssertNoError(err)
		if n != 1 {
			t.PrintDiff(1, n)
		}
	}

	if actual != sut {
		t.PrintDiff(sut, actual)
	}
}
