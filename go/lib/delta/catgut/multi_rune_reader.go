package catgut

import (
	"io"
	"strings"
)

func MakeMultiRuneReader(vs ...string) *MultiRuneReader {
	mrr := &MultiRuneReader{strings: vs}

	if len(vs) > 0 {
		mrr.sr.Reset(vs[0])
	}

	return mrr
}

type MultiRuneReader struct {
	sr         strings.Reader
	arrayIndex int
	strings    []string
}

func (mrr *MultiRuneReader) Reset(vs ...string) {
	mrr.strings = vs
	mrr.arrayIndex = 0
	mrr.sr.Reset(mrr.strings[0])
}

func (mrr *MultiRuneReader) ReadRune() (r rune, n int, err error) {
	for {
		r, n, err = mrr.sr.ReadRune()

		if err == io.EOF && mrr.arrayIndex < len(mrr.strings)-1 {
			mrr.arrayIndex++
			mrr.sr.Reset(mrr.strings[mrr.arrayIndex])
			continue
		}

		return r, n, err
	}
}

func (mrr *MultiRuneReader) UnreadRune() error {
	return mrr.sr.UnreadRune()
}
