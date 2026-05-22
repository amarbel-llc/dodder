package equals

import "github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"

func Iterable[T any](a, b interfaces.Iterable[T]) bool {
	return false
}
