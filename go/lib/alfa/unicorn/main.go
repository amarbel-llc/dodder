package unicorn

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

var IsSpace = unicode.IsSpace

func Not(f func(rune) bool) func(rune) bool {
	return func(r rune) bool {
		return !f(r)
	}
}

func CountRune(b []byte, r rune) (c int) {
	for i, w := 0, 0; i < len(b); i += w {
		runeValue, width := utf8.DecodeRune(b[i:])

		if runeValue != r {
			return c
		}

		c++
		w = width
	}

	return c
}

func CutNCharacters(data []byte, n int) ([]byte, []byte) {
	if n <= 0 {
		panic(fmt.Sprintf("n must be >= 0, but was %d", n))
	}

	if n == 0 {
		return nil, data
	}

	count := 0
	for i := 0; i < len(data); {
		if count == n {
			return data[:i], data[i:]
		}

		ch, size := utf8.DecodeRune(data[i:])

		if ch == utf8.RuneError {
			panic("invalid utf8 sequence")
		}

		i += size
		count++
	}

	return data, nil // Less than n runes in total
}
