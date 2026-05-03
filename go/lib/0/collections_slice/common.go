package collections_slice

import "unicode/utf8"

type (
	String = Slice[string]
	Byte   Slice[byte]
	Rune   Slice[rune]
)

func (slice Byte) ToSlice() Slice[byte] {
	return Slice[byte](slice)
}

func (slice Byte) ReadRune() (char rune, width int, err error) {
	return char, width, err
}

func (slice Rune) ToSlice() Slice[rune] {
	return Slice[rune](slice)
}

func (slice Rune) ReadRune() (char rune, width int, err error) {
	for _, char = range slice {
		break
	}

	width = utf8.RuneLen(char)

	return char, width, err
}
