package quiter

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

// returns a function that executes `AddString` for the given `adder`
func MakeFuncAddString[
	ELEMENT any,
	ELEMENT_PTR interfaces.SetterPtr[ELEMENT],
](
	adder interfaces.Adder[ELEMENT],
) interfaces.FuncSetString {
	return func(value string) (err error) {
		return AddString[ELEMENT, ELEMENT_PTR](adder, value)
	}
}

func Len(cs ...interfaces.Lenner) (n int) {
	for _, c := range cs {
		n += c.Len()
	}

	return n
}

func DerivedValues[E any, F any](
	c interfaces.Set[E],
	f interfaces.FuncTransform[E, F],
) (out []F, err error) {
	out = make([]F, 0, c.Len())

	for e := range c.All() {
		var e1 F

		if e1, err = f(e); err != nil {
			if errors.IsStopIteration(err) {
				err = nil
			} else {
				err = errors.Wrap(err)
			}

			return out, err
		}

		out = append(out, e1)
	}

	return out, err
}
