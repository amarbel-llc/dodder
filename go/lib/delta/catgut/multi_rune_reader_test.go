package catgut

import (
	"testing"

	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

func TestMultiRuneReader(t1 *testing.T) {
	t := ui.T{T: t1}

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
		err := mrr.UnreadRune()
		if err != nil {
			t.Errorf("%s", err)
		}
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
