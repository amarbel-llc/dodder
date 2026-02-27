package cmp

import "slices"

func BinarySearchFuncIndex[
	SLICE ~[]ELEMENT,
	ELEMENT any,
	TARGET any,
](
	slice SLICE,
	target TARGET,
	cmp func(ELEMENT, TARGET) Result,
) (int, bool) {
	return slices.BinarySearchFunc(
		slice,
		target,
		func(left ELEMENT, right TARGET) int {
			return cmp(left, right).GetCompareInt()
		},
	)
}

func BinarySearchFuncElement[
	SLICE ~[]ELEMENT,
	ELEMENT any,
	TARGET any,
](
	slice SLICE,
	target TARGET,
	cmp func(ELEMENT, TARGET) Result,
) (element ELEMENT, ok bool) {
	var index int

	index, ok = BinarySearchFuncIndex(
		slice,
		target,
		cmp,
	)

	if ok {
		element = slice[index]
	}

	return element, ok
}
