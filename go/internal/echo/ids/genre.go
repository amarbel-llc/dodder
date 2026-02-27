package ids

import (
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/ohio"
	"code.linenisgreat.com/dodder/go/internal/charlie/doddish"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
)

type Genre byte

func MakeGenreAll() Genre {
	return MakeGenre(genres.All()...)
}

func MakeGenre(vs ...genres.Genre) (s Genre) {
	s.Add(vs...)
	return s
}

func (genre Genre) IsEmpty() bool {
	return genre == 0
}

func (genre Genre) Equals(b Genre) bool {
	return genre == b
}

func (genre *Genre) Reset() {
	*genre = 0
}

func (genre *Genre) ResetWith(b Genre) {
	*genre = b
}

func (genre *Genre) Add(bs ...genres.Genre) {
	for _, b := range bs {
		*genre |= Genre(b.GetGenre().GetGenreBitInt())
	}
}

func (genre *Genre) Del(b domain_interfaces.GenreGetter) {
	*genre &= ^Genre(b.GetGenre().GetGenreBitInt())
}

func (genre Genre) Contains(b domain_interfaces.GenreGetter) bool {
	bg := Genre(b.GetGenre().GetGenreBitInt())
	return byte(genre&bg) == byte(bg)
}

func (genre Genre) ContainsOneOf(b domain_interfaces.GenreGetter) bool {
	bg := Genre(b.GetGenre().GetGenreBitInt())
	return genre&bg != 0
}

func (genre Genre) Slice() []genres.Genre {
	tg := genres.All()
	out := make([]genres.Genre, 0, len(tg))

	for _, g := range tg {
		if !genre.ContainsOneOf(g) {
			continue
		}

		out = append(out, g)
	}

	return out
}

func (genre Genre) String() string {
	sb := strings.Builder{}

	first := true

	for _, g := range genres.All() {
		if !genre.ContainsOneOf(g) {
			continue
		}

		if !first {
			sb.WriteRune(',')
		}

		sb.WriteString(g.String())
		first = false
	}

	return sb.String()
}

func (genre *Genre) AddString(v string) (err error) {
	var g genres.Genre

	if err = g.Set(v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	genre.Add(g)

	return err
}

func (genre *Genre) Set(v string) (err error) {
	v = strings.TrimSpace(v)
	v = strings.ToLower(v)

	for _, g := range strings.Split(v, ",") {
		if err = genre.AddString(g); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (genre *Genre) ReadFromBoxScanner(
	scanner *doddish.Scanner,
) (err error) {
	for scanner.Scan() {
		seq := scanner.GetSeq()

		switch {
		case seq.MatchAll(doddish.TokenTypeIdentifier):
			// etikett type zettel kasten konfig
			if err = genre.AddString(string(seq.At(0).Contents)); err != nil {
				err = errors.Wrap(err)
				return err
			}

		case seq.MatchAll(doddish.TokenMatcherOp(doddish.OpOr)):
			// ,
			continue

		case seq.MatchAll(doddish.TokenMatcherOp(doddish.OpAnd)):
			// " "
			scanner.Unscan()
			return err

		default:
			err = errors.ErrorWithStackf(
				"unsupported sequence: %q:%#v",
				seq,
				seq,
			)
			return err
		}
	}

	if err = scanner.Error(); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (genre Genre) Byte() byte {
	return byte(genre)
}

func (genre Genre) ReadByte() (byte, error) {
	return byte(genre), nil
}

func (genre *Genre) ReadFrom(r io.Reader) (n int64, err error) {
	var b [1]byte

	var n1 int
	n1, err = ohio.ReadAllOrDieTrying(r, b[:])
	n = int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	*genre = Genre(b[0])

	return n, err
}

func (genre *Genre) WriteTo(w io.Writer) (n int64, err error) {
	var b byte

	if b, err = genre.ReadByte(); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	var n1 int
	n1, err = ohio.WriteAllOrDieTrying(w, []byte{b})
	n = int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}
