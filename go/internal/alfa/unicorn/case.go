package unicorn

import (
	"unicode"
	"unicode/utf8"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
)

func CountCase(bites []byte) (lower, neither, upper int) {
	for char := range AllRunes(bites) {
		if unicode.IsUpper(char) {
			upper++
		} else if unicode.IsLower(char) {
			lower++
		} else {
			neither++
		}
	}

	return lower, neither, upper
}

func AllRunes(bites []byte) interfaces.Seq[rune] {
	return func(yield func(rune) bool) {
		for i := 0; i < len(bites); {
			char, size := utf8.DecodeRune(bites[i:])

			if !yield(char) {
				return
			}

			i += size
		}
	}
}

func AllRunesWithIndex(bites []byte) interfaces.Seq2[int, rune] {
	return func(yield func(int, rune) bool) {
		for i := 0; i < len(bites); {
			char, size := utf8.DecodeRune(bites[i:])

			if !yield(i, char) {
				return
			}

			i += size
		}
	}
}

func ToUpper(bites []byte) {
	for idx, char := range AllRunesWithIndex(bites) {
		char = unicode.ToUpper(char)
		utf8.EncodeRune(bites[idx:], char)
	}
}

func ToLower(bites []byte) {
	for idx, char := range AllRunesWithIndex(bites) {
		char = unicode.ToLower(char)
		utf8.EncodeRune(bites[idx:], char)
	}
}
