package quiter

import (
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
)

type StringerKeyer[
	T interfaces.Stringer,
] struct{}

func (sk StringerKeyer[T]) GetKey(e T) string {
	return e.String()
}

type StringerKeyerPtr[
	T interfaces.Stringer,
	TPtr interface {
		interfaces.Ptr[T]
		interfaces.Stringer
	},
] struct{}

func (sk StringerKeyerPtr[T, TPtr]) GetKey(e T) string {
	return e.String()
}

func (sk StringerKeyerPtr[T, TPtr]) GetKeyPtr(e TPtr) string {
	if e == nil {
		return ""
	}

	return e.String()
}
