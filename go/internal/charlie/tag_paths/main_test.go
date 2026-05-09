package tag_paths

import (
	"bytes"
	"testing"

	dodder_ui "code.linenisgreat.com/dodder/go/lib/alfa/ui"
	"code.linenisgreat.com/dodder/go/lib/bravo/catgut"
	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

func TestMain(m *testing.M) {
	dodder_ui.SetTesting()
	m.Run()
}

func TestReadWrite(t1 *testing.T) {
	t := ui.MakeT(t1)

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
		t.AssertEqual(b.Len(), int(n))
	}

	b.Reset()

	{
		n, err := sut.ReadFrom(b)
		t.AssertEOF(err)

		t.AssertEqual(b.Len(), int(n))

		t.AssertEqual(3, sut.Len())

		t.AssertTrue(sut[0].EqualsString("one"), "expected sut[0] to equal 'one'")

		t.AssertTrue(sut[1].EqualsString("two"), "expected sut[1] to equal 'two'")

		t.AssertTrue(sut[2].EqualsString("three"), "expected sut[2] to equal 'three'")
	}
}
