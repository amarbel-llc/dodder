package catgut

import (
	"bytes"
	"unicode"
	"unicode/utf8"
)

func MapTo(
	buffer *bytes.Buffer,
	s []byte,
	mapping func(r rune) rune,
) (n int) {
	buffer.Grow(len(s))
	available := buffer.AvailableBuffer()

	for i := 0; i < len(s); {
		wid := 1
		r := rune(s[i])

		if r >= utf8.RuneSelf {
			r, wid = utf8.DecodeRune(s[i:])
		}

		r = mapping(r)

		if r >= 0 {
			available = utf8.AppendRune(available, r)
		}

		i += wid
	}

	n, _ = buffer.Write(available)

	return n
}

func WriteLower(buffer *bytes.Buffer, s []byte) (n int) {
	isASCII, hasUpper := true, false

	for i := 0; i < len(s); i++ {
		c := s[i]

		if c >= utf8.RuneSelf {
			isASCII = false
			break
		}

		hasUpper = hasUpper || ('A' <= c && c <= 'Z')
	}

	buffer.Grow(len(s))

	if isASCII { // optimize for ASCII-only byte slices.
		b := buffer.AvailableBuffer()

		if !hasUpper {
			n, _ = buffer.Write(append(b, s...))
			return n
		}

		for i := 0; i < len(s); i++ {
			c := s[i]

			if 'A' <= c && c <= 'Z' {
				c += 'a' - 'A'
			}

			b = append(b, c)
		}

		n, _ = buffer.Write(b)

		return n
	}

	return MapTo(buffer, s, unicode.ToLower)
}
