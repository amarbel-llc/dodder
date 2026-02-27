package catgut

import (
	"io"
	"unicode/utf8"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
)

func MakeRingBufferRuneScanner(rb *RingBuffer) (s *RingBufferRuneScanner) {
	s = &RingBufferRuneScanner{}
	s.ResetWith(rb)

	return s
}

type RingBufferRuneScanner struct {
	rb *RingBuffer

	overlap                     [6]byte // three bytes from first, three bytes from second
	overlapFirst, overlapSecond int

	lastRuneSize int
}

func (s *RingBufferRuneScanner) ResetWith(rb *RingBuffer) {
	s.rb = rb
	s.resetSliceOnly()
}

func (s *RingBufferRuneScanner) resetSliceOnly() {
	s.overlap, s.overlapFirst, s.overlapSecond = s.rb.PeekReadable().Overlap()
}

func (s *RingBufferRuneScanner) ReadRune() (r rune, width int, err error) {
	if s.rb.Len() <= utf8.UTFMax {
		if _, err = s.rb.Fill(); err != nil {
			if err != io.EOF {
				err = errors.Wrap(err)
				return r, width, err
			}

			if s.rb.Len() >= 0 {
				err = nil
			} else {
				return r, width, err
			}
		}

		s.resetSliceOnly()
	}

	slice := s.rb.PeekReadable()
	first := slice.First()
	firstLen := len(first)

	switch {
	case firstLen >= utf8.UTFMax:
		r, width = utf8.DecodeRune(slice.First())

	case firstLen > 0:
		r, width = utf8.DecodeRune(s.overlap[s.overlapFirst-firstLen:])

	case len(slice.Second()) > 0:
		r, width = utf8.DecodeRune(slice.Second())

	default:
		err = io.EOF
		return r, width, err
	}

	s.lastRuneSize = width

	if r == utf8.RuneError {
		err = errInvalidRune
		return r, width, err
	}

	if width > 0 {
		s.rb.AdvanceRead(width)
	}

	return r, width, err
}

func (s *RingBufferRuneScanner) UnreadRune() (err error) {
	actuallyUnread := s.rb.Unread(s.lastRuneSize)

	if actuallyUnread != s.lastRuneSize {
		err = errors.ErrorWithStackf("tried to unread %d bytes but actually unread %d", s.lastRuneSize, actuallyUnread)
		return err
	}

	return err
}
