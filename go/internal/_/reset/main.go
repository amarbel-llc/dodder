package reset

import "golang.org/x/exp/constraints"

type (
	FuncReset[ELEMENT any]     func(ELEMENT)
	FuncResetWith[ELEMENT any] func(dst ELEMENT, src ELEMENT)
)

type resetter[ELEMENT any] struct {
	funcReset     FuncReset[ELEMENT]
	funcResetWith FuncResetWith[ELEMENT]
}

func MakeResetter[ELEMENT any](
	funcReset FuncReset[ELEMENT],
	funcResetWith FuncResetWith[ELEMENT],
) resetter[ELEMENT] {
	return resetter[ELEMENT]{
		funcReset:     funcReset,
		funcResetWith: funcResetWith,
	}
}

func (resetter resetter[ELEMENT]) Reset(element ELEMENT) {
	resetter.funcReset(element)
}

func (resetter resetter[ELEMENT]) ResetWith(dst, src ELEMENT) {
	resetter.funcResetWith(dst, src)
}

func Map[KEY constraints.Ordered, VALUE any](mapp map[KEY]VALUE) (out map[KEY]VALUE) {
	if mapp == nil {
		out = make(map[KEY]VALUE)
	} else {
		clear(mapp)
		out = mapp
	}

	return out
}

func Slice[ELEMENT any](in []ELEMENT) (out []ELEMENT) {
	if in == nil {
		out = make([]ELEMENT, 0)
	} else {
		out = in[:0]
	}

	return out
}
