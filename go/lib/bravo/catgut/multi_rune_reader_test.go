package catgut

import (
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

func TestMultiRuneReader(t1 *testing.T) {
	t := ui.MakeT(t1)

	input := []string{
		"wow",
		"nice",
		"hat",
	}

	mrr := MakeMultiRuneReader(input...)

	readOne := func(c rune) {
		r, n, err := mrr.ReadRune()

		if r != c || n != 1 || err != nil {
			t.Errorf("%c, %d, %s", r, n, err)
		}
	}

	unreadOne := func() {
		t.AssertNoError(mrr.UnreadRune())
	}

	readMany := func(cs ...rune) {
		for _, c := range cs {
			readOne(c)
		}
	}

	{
		mrr.Reset(input...)
		readMany('w', 'o', 'w', 'n', 'i', 'c', 'e', 'h', 'a', 't')
		unreadOne()
		readMany('t')
	}

	{
		mrr.Reset(input...)
		readMany('w', 'o', 'w', 'n')
		unreadOne()
		readMany('n')
	}

	{
		mrr.Reset(input...)
		readMany('w', 'o', 'w')
		unreadOne()
		readMany('w')
	}
}
