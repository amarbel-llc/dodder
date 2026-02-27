package pool

type BespokeResetter[T any] struct {
	FuncReset     func(T)
	FuncResetWith func(T, T)
}

func (br BespokeResetter[T]) Reset(e T) {
	br.FuncReset(e)
}

func (br BespokeResetter[T]) ResetWith(dst, src T) {
	br.FuncResetWith(dst, src)
}
