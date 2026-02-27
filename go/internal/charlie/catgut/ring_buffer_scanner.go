package catgut

import (
	"io"
)

type RingBufferScanner struct {
	sliceScanner SliceRuneScanner
	rb           *RingBuffer
}

func MakeRingBufferScanner(rb *RingBuffer) *RingBufferScanner {
	return &RingBufferScanner{
		rb: rb,
	}
}

var ErrNoMatch = newPkgError("no match")

func (s *RingBufferScanner) FirstMatch(
	mf func(rune) bool,
) (match Slice, offsetPlusMatch int, err error) {
	s.sliceScanner.ResetWith(s.rb.PeekReadable())

	if s.rb.PeekReadable().Len() == 0 {
		err = ErrBufferEmpty
		return match, offsetPlusMatch, err
	}

	startMatchOffset := -1

	var r rune
	var w int
	var ok bool

LOOP:
	for {
		r, w, ok = s.sliceScanner.Scan()

		if !ok {
			if err = s.sliceScanner.Error(); err != nil {
				if err == io.EOF {
					err = ErrBufferEmpty
				}
			}

			break
		}

		currentMatch := mf(r)

		switch {
		case currentMatch:
			if startMatchOffset < 0 {
				startMatchOffset = offsetPlusMatch
			}

		case !currentMatch && startMatchOffset >= 0:
			break LOOP
		}

		offsetPlusMatch += w
	}

	if startMatchOffset == -1 {
		s.rb.AdvanceRead(offsetPlusMatch)
		err = ErrNoMatch
		return match, offsetPlusMatch, err
	}

	match = s.rb.PeekReadable().Slice(startMatchOffset, offsetPlusMatch)

	return match, offsetPlusMatch, err
}
