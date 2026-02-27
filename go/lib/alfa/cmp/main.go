package cmp

import (
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

type Func[ELEMENT any] func(ELEMENT, ELEMENT) Result

// TODO remove
func MakeFuncFromEqualerAndLessor3EqualFirst[ELEMENT any](
	equaler interfaces.Equaler[ELEMENT],
	lessor interfaces.Lessor[ELEMENT],
) func(ELEMENT, ELEMENT) Result {
	return func(left, right ELEMENT) Result {
		if equaler.Equals(left, right) {
			return Equal
		} else if lessor.Less(left, right) {
			return Less
		} else {
			return Greater
		}
	}
}

// TODO remove
func MakeFuncFromEqualerAndLessor3LessFirst[ELEMENT any](
	equaler interfaces.Equaler[ELEMENT],
	lessor interfaces.Lessor[ELEMENT],
) func(ELEMENT, ELEMENT) Result {
	return func(left, right ELEMENT) Result {
		if lessor.Less(left, right) {
			return Less
		} else if equaler.Equals(left, right) {
			return Equal
		} else {
			return Greater
		}
	}
}

type Lesser[ELEMENT any] Func[ELEMENT]

func (lessor Lesser[T]) Less(left, right T) bool {
	return lessor(left, right).IsLess()
}

type Equaler[ELEMENT any] Func[ELEMENT]

func (equaler Equaler[T]) Equals(left, right T) bool {
	return equaler(left, right).IsEqual()
}
