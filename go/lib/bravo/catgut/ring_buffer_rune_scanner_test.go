package catgut

import (
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/alfa/pool"
	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
	"github.com/google/go-cmp/cmp"
)

func TestRingBufferRuneScanner(t1 *testing.T) {
	t := ui.MakeT(t1)
	input := `- [six/wow] seis`

	reader, repool := pool.GetStringReader(input)
	defer repool()
	rb := MakeRingBuffer(reader, 0)
	sut1 := MakeRingBufferRuneScanner(rb)
	sut2 := MakeRingBufferRuneScanner(rb)

	readOne := func(t *ui.T, s *RingBufferRuneScanner, c rune) {
		r, n, err := s.ReadRune()

		if r != c {
			t.Errorf("%s", cmp.Diff(string(c), string(r)))
		}

		if n != 1 {
			t.Errorf("%s", cmp.Diff(1, n))
		}

		t.AssertNoError(err)
	}

	unreadOne := func(t *ui.T, s *RingBufferRuneScanner) {
		t.AssertNoError(s.UnreadRune())
	}

	readMany := func(t *ui.T, s *RingBufferRuneScanner, cs ...rune) {
		for _, c := range cs {
			readOne(t.Skip(1), s, c)
		}
	}

	readMany(t.Skip(1), sut1, []rune("- [")...)
	unreadOne(t.Skip(1), sut1)
	readMany(t.Skip(1), sut2, []rune("[six")...)
	readMany(t.Skip(1), sut1, []rune("/wow]")...)
	unreadOne(t.Skip(1), sut1)
	readMany(t.Skip(1), sut2, []rune("]")...)
}
