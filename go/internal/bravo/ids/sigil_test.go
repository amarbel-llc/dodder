package ids

import (
	"bytes"
	"testing"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

func TestSigilContains(t1 *testing.T) {
	t := ui.MakeT(t1)

	sut := SigilAll

	t.AssertTrue(sut.ContainsOneOf(SigilLatest), "expected SigilAll to contain SigilLatest")

	sut = SigilLatest
	sut.Add(SigilHidden)

	t.AssertTrue(sut.ContainsOneOf(SigilLatest), "expected sut to contain SigilTail")
	t.AssertTrue(sut.ContainsOneOf(SigilHidden), "expected sut to contain SigilHidden")
	t.AssertTrue(sut.ContainsOneOf(sut), "expected sut to contain sut")
	t.AssertTrue(sut.Contains(sut), "expected sut to contain sut")

	other := SigilHistory
	other.Add(SigilHidden)

	t.AssertTrue(sut.ContainsOneOf(other), "expected sut to contain one ofother")
	t.AssertFalse(sut.Contains(other), "expected sut not to contain other")
	t.AssertFalse(sut.ContainsOneOf(SigilExternal), "expected sut not to contain SigilCwd")
}

func TestSigilReadWrite(t1 *testing.T) {
	t := ui.MakeT(t1)

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
