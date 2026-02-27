package doddish

import (
	"testing"

	"code.linenisgreat.com/dodder/go/internal/bravo/ui"
	"github.com/google/go-cmp/cmp"
)

func TestSeqRuneScanner(t1 *testing.T) {
	t := ui.T{T: t1}

	seq := makeTestSeq(
		TokenTypeIdentifier, "uno",
		TokenTypeOperator, "/",
		TokenTypeIdentifier, "dos",
	)

	sut := &SeqRuneScanner{Seq: makeSeqFromTestSeq(seq)}

	readOne := func(t *ui.T, s *SeqRuneScanner, c rune) {
		r, n, err := s.ReadRune()

		if r != c {
			t.Errorf("%s", cmp.Diff(string(c), string(r)))
		}

		if n != 1 {
			t.Errorf("%s", cmp.Diff(1, n))
		}

		if err != nil {
			t.Errorf("%s", cmp.Diff(nil, err))
		}
	}

	unreadOne := func(t *ui.T, s *SeqRuneScanner) {
		err := s.UnreadRune()
		if err != nil {
			t.Errorf("%s", err)
		}
	}

	readMany := func(t *ui.T, s *SeqRuneScanner, cs ...rune) {
		for _, c := range cs {
			readOne(t.Skip(1), s, c)
		}
	}

	t.AssertError(sut.UnreadRune())
	readMany(t.Skip(1), sut, []rune("uno")...)
	unreadOne(t.Skip(1), sut)
	readMany(t.Skip(1), sut, []rune("o/")...)
	unreadOne(t.Skip(1), sut)
	readMany(t.Skip(1), sut, []rune("/dos")...)

	sut = &SeqRuneScanner{Seq: makeSeqFromTestSeq(seq)}
	readMany(t.Skip(1), sut, []rune("uno/dos")...)
	unreadOne(t.Skip(1), sut)
	readMany(t.Skip(1), sut, []rune("s")...)
	{
		_, _, err := sut.ReadRune()
		t.AssertError(err)
	}
}
