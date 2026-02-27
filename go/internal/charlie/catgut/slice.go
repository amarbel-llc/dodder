package catgut

import (
	"bytes"
	"io"
	"strings"
)

type SliceBytes struct {
	Slice
	Bytes []byte
}

type Slice struct {
	start int64
	data  [2][]byte
}

func (a Slice) SliceBytes() SliceBytes {
	return SliceBytes{
		Slice: a,
		Bytes: a.Bytes(),
	}
}

func (a Slice) FirstByte() byte {
	switch {
	case a.LenFirst() > 0:
		return a.First()[0]

	case a.LenSecond() > 0:
		return a.Second()[0]

	default:
		panic("FirstByte called on empty slice")
	}
}

func (a Slice) LastByte() byte {
	switch {
	case a.LenSecond() > 0:
		return a.Second()[a.LenSecond()-1]

	case a.LenFirst() > 0:
		return a.First()[a.LenFirst()-1]

	default:
		panic("LastByte called on empty slice")
	}
}

func (a Slice) Slice(left, right int) (b Slice) {
	lastIdx := a.Len()

	if left < 0 || right < 0 || right < left || left > lastIdx || right > lastIdx {
		panic(errInvalidSliceRange{left, right})
	}

	b.start = a.start + int64(left)

	lenFirst := len(a.First())

	switch {
	case right < lenFirst:
		b.data[0] = a.data[0][left:right]

	case left > lenFirst:
		b.data[0] = a.data[1][left-lenFirst : right-lenFirst]

	default:
		b.data[0] = a.data[0][left:]
		b.data[1] = a.data[1][:right-lenFirst]
	}

	return b
}

func (rs Slice) Overlap() (o [6]byte, first, second int) {
	firstEnd := rs.First()

	if len(firstEnd) > 3 {
		firstEnd = firstEnd[len(firstEnd)-3:]
	}

	secondEnd := rs.Second()

	if len(secondEnd) > 3 {
		secondEnd = secondEnd[:3]
	}

	first = copy(o[:3], firstEnd)
	second = copy(o[first:], secondEnd)

	return o, first, second
}

func (rs Slice) ReadFrom(r io.Reader) (n int64, err error) {
	var loc int

	for n < int64(rs.LenFirst()) {
		loc, err = r.Read(rs.First()[n:])
		n += int64(loc)
		if err != nil {
			return n, err
		}
	}

	for n < int64(rs.LenSecond()-rs.LenFirst()) {
		loc, err = r.Read(rs.Second()[n-int64(rs.LenFirst()):])
		n += int64(loc)
		if err != nil {
			return n, err
		}
	}

	return n, err
}

func (rs Slice) WriteTo(w io.Writer) (n int64, err error) {
	var n1 int

	n1, err = w.Write(rs.First())
	n += int64(n1)

	if err != nil {
		return n, err
	}

	n1, err = w.Write(rs.Second())
	n += int64(n1)

	return n, err
}

func (rs Slice) BytesBetween(left, right int) []byte {
	if left > right {
		panic(newPkgError("left greater than right"))
	}

	switch {
	case right < len(rs.data[0]):
		return rs.data[0][left:right]

	case left >= len(rs.data[0]):
		return rs.data[1][left-len(rs.data[0]) : right-len(rs.data[0])]

	default:
		b := make([]byte, right-left)
		n := copy(b, rs.data[0][left:])
		copy(b[n:], rs.data[1][:right-len(rs.data)])
		return rs.data[1][left-len(rs.data[0]) : right-len(rs.data[0])]
	}
}

func (rs Slice) Bytes() []byte {
	switch {
	case len(rs.First()) == rs.Len():
		return rs.First()

	case len(rs.Second()) == rs.Len():
		return rs.Second()

	default:
		var b bytes.Buffer
		b.Grow(rs.Len())
		b.Write(rs.First())
		b.Write(rs.Second())
		return b.Bytes()
	}
}

func (rs Slice) String() string {
	var s strings.Builder
	s.Grow(rs.Len())
	s.Write(rs.First())
	s.Write(rs.Second())
	return s.String()
}

func (rs Slice) Equal(b []byte) bool {
	if len(b) < rs.Len() {
		return false
	}

	c := 0

	for _, v := range rs.First() {
		if b[c] != v {
			return false
		}

		c++
	}

	for _, v := range rs.Second() {
		if b[c] != v {
			return false
		}

		c++
	}

	return true
}

func (rs Slice) Compare(b []byte) (c int) {
	if len(b) < len(rs.First()) {
		c = bytes.Compare(b, rs.First())
	} else {
		c = bytes.Compare(b[:len(rs.First())], rs.First())
	}

	if len(b) > len(rs.First()) && c == 0 {
		c = bytes.Compare(b[len(rs.First()):], rs.Second())
	}

	return c
}

func (rs Slice) Start() int64 {
	return rs.start
}

func (rs Slice) LenFirst() int {
	return len(rs.data[0])
}

func (rs Slice) LenSecond() int {
	return len(rs.data[1])
}

func (rs Slice) First() []byte {
	return rs.data[0]
}

func (rs Slice) Second() []byte {
	return rs.data[1]
}

func (rs Slice) IsEmpty() bool {
	return rs.Len() == 0
}

func (rs Slice) Len() int {
	return len(rs.First()) + len(rs.Second())
}

func (rs Slice) Cut(b byte) (before, after Slice, ok bool) {
	for i, v := range rs.First() {
		if v == b {
			before = Slice{
				start: rs.start,
				data: [2][]byte{
					rs.First()[:i],
					nil,
				},
			}

			after = Slice{
				start: rs.start + int64(1+before.Len()),
				data: [2][]byte{
					rs.First()[i+1:],
					rs.Second(),
				},
			}

			ok = true

			return before, after, ok
		}
	}

	for i, v := range rs.Second() {
		if v == b {
			before = Slice{
				start: rs.start,
				data: [2][]byte{
					rs.First(),
					rs.Second()[:i],
				},
			}

			after = Slice{
				start: rs.start + int64(1+before.Len()),
				data: [2][]byte{
					rs.Second()[i+1:],
					nil,
				},
			}

			ok = true

			return before, after, ok
		}
	}

	return before, after, ok
}

func (rs Slice) SliceUptoButExcluding(b byte) (s Slice, ok bool) {
	for i, v := range rs.First() {
		if v == b {
			s = Slice{
				start: rs.start,
				data: [2][]byte{
					rs.First()[:i],
					nil,
				},
			}

			ok = true

			return s, ok
		}
	}

	for i, v := range rs.Second() {
		if v == b {
			s = Slice{
				start: rs.start,
				data: [2][]byte{
					rs.First(),
					rs.Second()[:i],
				},
			}

			ok = true

			return s, ok
		}
	}

	return s, ok
}

func (rs Slice) SliceUptoAndIncluding(b byte) (s Slice, ok bool) {
	for i, v := range rs.First() {
		if v == b {
			s = Slice{
				start: rs.start,
				data: [2][]byte{
					rs.First()[:i+1],
					nil,
				},
			}

			ok = true

			return s, ok
		}
	}

	for i, v := range rs.Second() {
		if v == b {
			s = Slice{
				start: rs.start,
				data: [2][]byte{
					rs.First(),
					rs.Second()[:i+1],
				},
			}

			ok = true

			return s, ok
		}
	}

	return s, ok
}
