package cmp

import (
	"unicode/utf8"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
)

func CompareUTF8Bytes(left, right []byte, partial bool) Result {
	return CompareUTF8(
		ComparableBytes(left),
		ComparableBytes(right),
		partial,
	)
}

func CompareUTF8BytesAndString(left []byte, right string, partial bool) Result {
	return CompareUTF8(
		ComparableBytes(left),
		ComparableString(right),
		partial,
	)
}

func CompareUTF8StringAndBytes(left string, right []byte, partial bool) Result {
	return CompareUTF8(
		ComparableString(left),
		ComparableBytes(right),
		partial,
	)
}

func CompareUTF8String(left, right string, partial bool) Result {
	return CompareUTF8(
		ComparableString(left),
		ComparableString(right),
		partial,
	)
}

func CompareUTF8[
	LEFT interfaces.Comparable[LEFT],
	RIGHT interfaces.Comparable[RIGHT],
](
	left LEFT,
	right RIGHT,
	partial bool,
) Result {
	lenLeft, lenRight := left.Len(), right.Len()

	// TODO remove?
	switch {
	case lenLeft == 0 && lenRight == 0:
		return Equal

	case lenLeft == 0:
		return Less

	case lenRight == 0:
		return Greater
	}

	for {
		lenLeft, lenRight := left.Len(), right.Len()

		switch {
		case lenLeft == 0 && lenRight == 0:
			return Equal

		case lenLeft == 0:
			if partial && lenRight <= lenLeft {
				return Equal
			} else {
				return Less
			}

		case lenRight == 0:
			if partial {
				return Equal
			} else {
				return Greater
			}
		}

		var runeLeft, runeRight rune

		{
			var width int

			runeLeft, width = left.DecodeRune()
			left = left.Shift(width)

			if runeLeft == utf8.RuneError {
				panic("not a valid utf8 string")
			}
		}

		{
			var width int

			runeRight, width = right.DecodeRune()
			right = right.Shift(width)

			if runeRight == utf8.RuneError {
				panic("not a valid utf8 string")
			}
		}

		if runeLeft < runeRight {
			return Less
		} else if runeLeft > runeRight {
			return Greater
		}
	}
}

type ComparableBytes []byte

func (slice ComparableBytes) Len() int {
	return len(slice)
}

func (slice ComparableBytes) DecodeRune() (char rune, width int) {
	char, width = utf8.DecodeRune(slice)
	return char, width
}

func (slice ComparableBytes) Shift(amount int) ComparableBytes {
	return slice[amount:]
}

type ComparableString string

func (comparer ComparableString) Len() int {
	return len(comparer)
}

func (comparer ComparableString) Shift(start int) ComparableString {
	return comparer[start:]
}

func (comparer ComparableString) DecodeRune() (char rune, width int) {
	for _, char = range comparer {
		break
	}

	width = utf8.RuneLen(char)

	return char, width
}
