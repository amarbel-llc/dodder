package tag_paths

import (
	"io"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/lib/bravo/ohio"
)

//go:generate stringer -type=Type
type Type byte

// describe these
const (
	TypeDirect = Type(iota)
	TypeSuper
	TypeIndirect
	TypeSelf
	TypeUnknown
)

// TODO determine if this should include type self
func (tipe Type) IsDirectOrSelf() bool {
	switch tipe {
	case TypeDirect, TypeSelf:
		return true

	default:
		return false
	}
}

func (tipe *Type) SetDirect() {
	*tipe = TypeDirect
}

func (tipe Type) ReadByte() (byte, error) {
	return byte(tipe), nil
}

func (tipe *Type) ReadFrom(r io.Reader) (n int64, err error) {
	var b [1]byte

	var n1 int
	n1, err = ohio.ReadAllOrDieTrying(r, b[:])
	n = int64(n1)

	if err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	*tipe = Type(b[0])

	return n, err
}

func (tipe Type) WriteTo(w io.Writer) (n int64, err error) {
	var b byte

	if b, err = tipe.ReadByte(); err != nil {
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
