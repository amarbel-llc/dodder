package equals

import "code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"

func Iterable[T any](a, b interfaces.Iterable[T]) bool {
	return false
}
