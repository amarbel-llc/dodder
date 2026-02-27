package ids

import (
	"io"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/ohio"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
)

type Sigil byte

const (
	SigilUnknown = Sigil(iota)
	SigilLatest  = Sigil(1 << iota)
	SigilHistory
	SigilExternal
	SigilHidden

	SigilMax
	SigilAll = Sigil(^byte(0))
)

var (
	mapRuneToSigil = map[rune]Sigil{
		':': SigilLatest,
		'+': SigilHistory,
		'.': SigilExternal,
		'?': SigilHidden,
	}

	mapSigilToRune = map[Sigil]rune{
		SigilLatest:   ':',
		SigilHistory:  '+',
		SigilExternal: '.',
		SigilHidden:   '?',
	}
)

func SigilFieldFunc(c rune) (ok bool) {
	_, ok = mapRuneToSigil[c]
	return ok
}

func MakeSigil(vs ...Sigil) (s Sigil) {
	for _, v := range vs {
		s.Add(v)
	}

	return s
}

func (sigil Sigil) GetGenre() domain_interfaces.Genre {
	return genres.Unknown
}

func (sigil Sigil) Equals(b Sigil) bool {
	return sigil == b
}

func (sigil Sigil) IsEmpty() bool {
	return sigil == SigilUnknown
}

func (sigil *Sigil) Reset() {
	*sigil = SigilLatest
}

func (sigil *Sigil) ResetWith(b Sigil) {
	*sigil = b
}

func (sigil *Sigil) Add(b Sigil) {
	*sigil |= b
}

func (sigil *Sigil) Del(b Sigil) {
	*sigil &= ^b
}

func (sigil Sigil) Contains(b Sigil) bool {
	return byte(sigil&b) == byte(b)
}

func (sigil Sigil) ContainsOneOf(b Sigil) bool {
	return sigil&b != 0
}

func (sigil Sigil) IsLatestOrUnknown() bool {
	return sigil == SigilLatest || sigil == SigilUnknown ||
		sigil == SigilLatest|SigilUnknown
}

func (sigil Sigil) IncludesLatest() bool {
	return sigil.ContainsOneOf(SigilLatest) ||
		sigil.ContainsOneOf(SigilHistory) ||
		sigil == 0
}

func (sigil Sigil) IncludesHistory() bool {
	return sigil.ContainsOneOf(SigilHistory)
}

func (sigil Sigil) IncludesExternal() bool {
	return sigil.ContainsOneOf(SigilExternal)
}

func (sigil Sigil) IncludesHidden() bool {
	return sigil.ContainsOneOf(SigilHidden) ||
		sigil.ContainsOneOf(SigilExternal)
}

func (sigil Sigil) String() string {
	sb := strings.Builder{}

	for s := SigilLatest; s <= SigilMax; s++ {
		if sigil&s != 0 {
			r, ok := mapSigilToRune[s]

			if !ok {
				continue
			}

			sb.WriteRune(r)
		}
	}

	return sb.String()
}

func (sigil *Sigil) SetByte(r byte) (err error) {
	if v, ok := mapRuneToSigil[rune(r)]; ok {
		sigil.Add(v)
	} else {
		err = errors.Wrap(errInvalidSigil(r))
		return err
	}

	return err
}

func (sigil *Sigil) Set(v string) (err error) {
	v = strings.TrimSpace(v)
	v = strings.ToLower(v)

	els := v

	for _, v1 := range els {
		if _, ok := mapRuneToSigil[v1]; ok {
			sigil.Add(mapRuneToSigil[v1])
		} else {
			err = errors.Wrap(errInvalidSigil(v))
			return err
		}
	}

	return err
}

func (sigil Sigil) Byte() byte {
	if sigil == SigilUnknown {
		return byte(SigilLatest)
	} else {
		return byte(sigil)
	}
}

func (sigil Sigil) ReadByte() (byte, error) {
	return byte(sigil), nil
}

func (sigil *Sigil) ReadFrom(r io.Reader) (n int64, err error) {
	var b [1]byte

	var n1 int
	n1, err = ohio.ReadAllOrDieTrying(r, b[:])
	n = int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	*sigil = Sigil(b[0])

	return n, err
}

func (sigil *Sigil) WriteTo(w io.Writer) (n int64, err error) {
	var b byte

	if b, err = sigil.ReadByte(); err != nil {
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
