package values

import (
	"strconv"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

type IntEqualer struct{}

func (_ IntEqualer) Equals(a, b *Int) bool {
	return a.Int() == b.Int()
}

func (_ IntEqualer) EqualsPtr(a, b *Int) bool {
	return a.Int() == b.Int()
}

type IntLessor struct{}

func (_ IntLessor) Less(a, b *Int) bool {
	return a.Int() < b.Int()
}

func (_ IntLessor) LessPtr(a, b *Int) bool {
	return a.Int() < b.Int()
}

type IntResetter struct{}

func (_ IntResetter) Reset(a *Int) {
	*a = 0
}

func (_ IntResetter) ResetWith(a *Int, b *Int) {
	*a = *b
}

type Int int

func MakeInt(i int) *Int {
	j := Int(i)
	return &j
}

func (a Int) Equals(b Int) (ok bool) {
	ok = a.Int() == b.Int()

	return ok
}

func (a Int) Less(b Int) (ok bool) {
	ok = a.Int() < b.Int()
	return ok
}

func (iv *Int) Reset() {
	*iv = Int(0)
}

func (iv *Int) ResetWith(b Int) {
	iv.SetInt(b.Int())
}

func (iv *Int) SetInt(i int) {
	*iv = Int(i)
}

func (iv *Int) Set(v string) (err error) {
	var i int

	if i, err = strconv.Atoi(v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	iv.SetInt(i)

	return err
}

func (iv Int) String() string {
	return strconv.Itoa(int(iv))
}

func (iv Int) Int() int {
	return int(iv)
}
