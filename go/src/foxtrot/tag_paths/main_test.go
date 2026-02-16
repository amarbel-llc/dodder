package tag_paths

import (
	"bytes"
	"testing"

	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/catgut"
)

func TestMain(m *testing.M) {
	ui.SetTesting()
	m.Run()
}

func TestReadWrite(t1 *testing.T) {
	t := ui.T{T: t1}

	b := new(bytes.Buffer)
	var sut Path

	one, _ := catgut.MakeFromString("one")
	sut.Add(one)
	two, _ := catgut.MakeFromString("two")
	sut.Add(two)
	three, _ := catgut.MakeFromString("three")
	sut.Add(three)

	{
		n, err := sut.WriteTo(b)
		t.AssertNoError(err)
		if int(n) != b.Len() {
			t.PrintDiff(b.Len(), n)
		}
	}

	b.Reset()

	{
		n, err := sut.ReadFrom(b)
		t.AssertEOF(err)

		if int(n) != b.Len() {
			t.PrintDiff(b.Len(), n)
		}

		if sut.Len() != 3 {
			t.PrintDiff(3, sut.Len())
		}

		if !sut[0].EqualsString("one") {
			t.PrintDiff("one", sut[0])
		}

		if !sut[1].EqualsString("two") {
			t.PrintDiff("two", sut[1])
		}

		if !sut[2].EqualsString("three") {
			t.PrintDiff("three", sut[2])
		}
	}
}
