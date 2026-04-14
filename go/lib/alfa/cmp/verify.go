package cmp

import (
	"fmt"

	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

type EqualerVerify[ELEMENT any] []interfaces.Equaler[ELEMENT]

func (equaler EqualerVerify[T]) Equals(left, right T) bool {
	primaryResult := equaler[0].Equals(left, right)

	for _, other := range equaler[1:] {
		otherResult := other.Equals(left, right)

		if primaryResult != otherResult {
			panic(
				fmt.Sprintf(
					"expected %t but got %t",
					primaryResult,
					otherResult,
				),
			)
		}
	}

	return primaryResult
}

type LesserVerify[ELEMENT any] []interfaces.Lessor[ELEMENT]

func (lessor LesserVerify[T]) Less(left, right T) bool {
	primaryResult := lessor[0].Less(left, right)

	for _, other := range lessor[1:] {
		otherResult := other.Less(left, right)

		if primaryResult != otherResult {
			panic(
				fmt.Sprintf(
					"expected %t but got %t",
					primaryResult,
					otherResult,
				),
			)
		}
	}

	return primaryResult
}
