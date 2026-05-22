package doddish

import (
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

func TestSeqRuneScanner(t1 *testing.T) {
	t := ui.MakeT(t1)

	seq := makeTestSeq(
		TokenTypeIdentifier, "uno",
		TokenTypeOperator, "/",
		TokenTypeIdentifier, "dos",
	)

	sut := &SeqRuneScanner{Seq: makeSeqFromTestSeq(seq)}

	readOne := func(t *ui.T, s *SeqRuneScanner, c rune) {
		r, n, err := s.ReadRune()

		t.AssertEqual(string(c), string(r))
		t.AssertEqual(1, n)
		t.AssertNoError(err)
	}

	unreadOne := func(t *ui.T, s *SeqRuneScanner) {
		err := s.UnreadRune()
		t.AssertNoError(err)
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
