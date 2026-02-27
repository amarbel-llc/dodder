package catgut

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"unicode"

	"code.linenisgreat.com/dodder/go/lib/alfa/pool"
	"code.linenisgreat.com/dodder/go/lib/alfa/unicorn"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

func TestRingBufferReader(t1 *testing.T) {
	t := ui.T{T: t1}
	expected := "all that content"
	reader, repool := pool.GetStringReader(expected)
	defer repool()
	sut := MakeRingBuffer(reader, 0)

	var sb strings.Builder

	n, err := io.Copy(&sb, sut)
	t.AssertNoError(err)

	if n != int64(len(expected)) {
		t.Errorf("expected %d but got %d", len(expected), n)
	}

	actual := sb.String()

	if actual != expected {
		t.PrintDiff(expected, actual)
	}
}

func TestRingBufferEmpty(t1 *testing.T) {
	t := ui.T{T: t1}
	sut := MakeRingBuffer(nil, 10)

	{
		actual := sut.Len()

		if sut.Len() != 0 {
			t.Errorf("expected %d but got %d", 0, actual)
		}
	}

	{
		n, err := sut.Write([]byte("test"))

		if n != 4 {
			t.Errorf("expected %d but got %d", 4, n)
		}

		t.AssertNoError(err)

		{
			expected := 4
			actual := sut.Len()

			if expected != actual {
				t.Errorf("expected %d but got %d", expected, actual)
			}
		}
	}

	// {
	// 	start, end := sut.Find([]byte("test"))

	// 	if start != 0 {
	// 		t.Errorf("expected %d but got %d", 0, start)
	// 	}

	// 	if end != 3 {
	// 		t.Errorf("expected %d but got %d", 3, end)
	// 	}
	// }

	{
		b := make([]byte, 4)
		n, err := sut.Read(b)

		if n != 4 {
			t.Errorf("expected %d but got %d", 4, n)
		}

		t.AssertEOF(err)

		actual := string(b)

		if actual != "test" {
			t.Errorf("expected %q but got %q", "test", actual)
		}
	}

	// {
	// 	t.Logf("%#v", sut)
	// 	start, end := sut.Find([]byte("t"))

	// 	if start != -1 {
	// 		t.Errorf("expected start %d but got %d", -1, start)
	// 	}

	// 	if end != -1 {
	// 		t.Errorf("expected end %d but got %d", -1, end)
	// 	}
	// }
}

func TestRingBufferEmptyFindFromStartAndAdvance(t1 *testing.T) {
	t := ui.T{T: t1}
	sut := MakeRingBuffer(nil, 10)

	{
		actual := sut.Len()

		if sut.Len() != 0 {
			t.Errorf("expected %d but got %d", 0, actual)
		}
	}

	{
		n, err := sut.Write([]byte("test"))

		if n != 4 {
			t.Errorf("expected %d but got %d", 4, n)
		}

		t.AssertNoError(err)

		{
			expected := 4
			actual := sut.Len()

			if expected != actual {
				t.Errorf("expected %d but got %d", expected, actual)
			}
		}
	}
}

func TestRingBufferEmptyTooBig(t1 *testing.T) {
	t := ui.T{T: t1}
	sut := MakeRingBuffer(nil, 5)

	for i := 0; i < 11; i++ {
		{
			n, err := sut.Write([]byte("test"))

			if n != 4 {
				t.Errorf("expected %d but got %d", 4, n)
			}

			t.AssertNoError(err)
		}

		{
			b := make([]byte, 4)
			n, err := sut.Read(b)

			if n != 4 {
				t.Errorf("expected %d but got %d", 4, n)
			}

			t.AssertEOF(err)

			{
				actual := string(b[:n])
				expected := "test"

				if actual != expected {
					t.Errorf("expected %q but got %q", expected, actual)
				}
			}
		}
	}

	// {
	// 	sut.Write([]byte("test"))
	// 	start, end := sut.Find([]byte("test"))

	// 	if start != 48 {
	// 		t.Errorf("expected %d but got %d", 48, start)
	// 	}

	// 	if end != -1 {
	// 		t.Errorf("expected %d but got %d", -1, end)
	// 	}
	// }
}

func TestRingBufferEmptyTooSmall(t1 *testing.T) {
	t := ui.T{T: t1}
	sut := MakeRingBuffer(bytes.NewBuffer(nil), 3)

	{
		n, err := sut.Write([]byte("teal"))

		if n != 3 {
			t.Errorf("expected %d but got %d", 4, n)
		}

		t.AssertEOF(err)

		if sut.Len() != 3 {
			t.Errorf("expected len 3 but got %d", sut.Len())
		}
	}

	{
		n, err := sut.Write([]byte("teal"))

		if n != 0 {
			t.Errorf("expected %d but got %d", 4, n)
		}

		t.AssertEOF(err)

		if sut.Len() != 3 {
			t.Errorf("expected len 3 but got %d", sut.Len())
		}
	}

	{
		b := make([]byte, 4)
		n, err := sut.Read(b)

		{
			expected := 3
			if n != expected {
				t.Errorf("expected %d but got %d", expected, n)
			}

			if sut.Len() != 0 {
				t.Errorf("expected len 0 but got %d", sut.Len())
			}
		}

		t.AssertEOF(err)

		actual := string(b[:n])
		expected := "tea"

		if actual != expected {
			t.Errorf("expected %q but got %q", expected, actual)
		}
	}

	{
		b := make([]byte, 4)
		n, err := sut.Read(b)

		{
			expected := 0
			if n != expected {
				t.Errorf("expected %d but got %d", expected, n)
			}
		}

		t.AssertEOF(err)
	}
}

func TestRingBufferDefault(t1 *testing.T) {
	t := ui.T{T: t1}
	t2 := t.Skip(1)
	sut := MakeRingBuffer(nil, 0)

	one_5 := make([]byte, 2730)
	half := make([]byte, 2048)

	l := 0

	write := func() {
		n, err := sut.Write(one_5)

		if n != len(one_5) {
			t2.Errorf("expected %d but got %d", len(one_5), n)
		}

		l += n

		t2.AssertNoError(err)

		if sut.Len() != l {
			t2.Errorf("expected len %d but got %d", l, sut.Len())
		}
	}

	read := func() {
		n, err := sut.Read(half)

		if n != len(half) {
			t2.Errorf("expected %d but got %d", len(half), n)
		}

		l -= n

		t2.AssertNoError(err)

		if sut.Len() != l {
			t2.Errorf("expected len %d but got %d", l, sut.Len())
		}
	}

	write()
	read()
	write()
	read()
	write()
	read()
}

func TestRingBufferDefaultReadFrom(t1 *testing.T) {
	t := ui.T{T: t1}
	t.SkipTest()

	one_5 := bytes.NewBuffer(make([]byte, 2730))
	sut := MakeRingBuffer(one_5, 0)

	half := make([]byte, 2048)

	l := 0
	t2 := t.Skip(1)

	write := func() {
		n, err := sut.Fill()
		one_5 = bytes.NewBuffer(make([]byte, 2730))

		if int(n) != one_5.Len() {
			t2.Errorf("expected %d but got %d", one_5.Len(), n)
		}

		l += int(n)

		t2.AssertNoError(err)

		if sut.Len() != l {
			t2.Errorf("expected len %d but got %d", l, sut.Len())
		}
	}

	read := func() {
		n, err := sut.Read(half)

		if n != len(half) {
			t2.Errorf("expected %d but got %d", len(half), n)
		}

		l -= n

		t2.AssertNoError(err)

		if sut.Len() != l {
			t2.Errorf("expected len %d but got %d", l, sut.Len())
		}
	}

	write()
	read()
	write()
	read()
	write()
	read()
}

func TestRingBufferPeekUpto2(t1 *testing.T) {
	t := ui.T{T: t1}
	reader, repool := pool.GetStringReader("test with words")
	defer repool()
	sut := MakeRingBuffer(reader, 0)

	{
		readable, err := sut.PeekUptoAndIncluding(' ')
		t.AssertNoError(err)
		t.AssertEqualStrings("test ", readable.String())
		sut.AdvanceRead(readable.Len())
	}

	{
		readable, err := sut.PeekUptoAndIncluding(' ')
		t.AssertNoError(err)
		t.AssertEqualStrings("with ", readable.String())
		sut.AdvanceRead(readable.Len())
	}

	{
		readable, _ := sut.PeekUptoAndIncluding(' ')
		// readable, err := sut.PeekUptoAndIncluding(' ')
		// TODO fix issue with not found error not matching
		// t.AssertErrorEquals(err, collections.ErrNotFound)
		t.AssertEqualStrings("words", sut.PeekReadable().String())
		sut.AdvanceRead(readable.Len())
	}
}

func TestRingBufferAdvanceToFirstMatch(t1 *testing.T) {
	t := ui.T{T: t1}
	reader, repool := pool.GetStringReader(" test with words")
	defer repool()
	rb := MakeRingBuffer(reader, 0)
	sut := MakeRingBufferScanner(rb)

	{
		readable, _, err := sut.FirstMatch(unicorn.Not(unicode.IsSpace))
		t.AssertErrorEquals(ErrBufferEmpty, err)
		t.AssertEqualStrings("", readable.String())
	}

	rb.Fill()

	{
		match, offsetPlusMatch, err := sut.FirstMatch(unicorn.Not(unicode.IsSpace))
		t.AssertNoError(err)
		t.AssertEqualStrings("test", match.String())
		rb.AdvanceRead(offsetPlusMatch)
		t.AssertEqualStrings(" with words", rb.PeekReadable().String())
	}

	{
		match, offsetPlusMatch, err := sut.FirstMatch(unicorn.Not(unicode.IsSpace))
		t.AssertNoError(err)
		t.AssertEqualStrings("with", match.String())
		rb.AdvanceRead(offsetPlusMatch)
		t.AssertEqualStrings(" words", rb.PeekReadable().String())
	}

	{
		match, offsetPlusMatch, err := sut.FirstMatch(unicorn.Not(unicode.IsSpace))
		t.AssertErrorEquals(ErrBufferEmpty, err)
		t.AssertEqualStrings("words", match.String())
		rb.AdvanceRead(offsetPlusMatch)
		t.AssertEqualStrings("", rb.PeekReadable().String())
	}

	{
		readable, offsetPlusMatch, err := sut.FirstMatch(unicorn.Not(unicode.IsSpace))
		t.AssertErrorEquals(ErrBufferEmpty, err)
		rb.AdvanceRead(offsetPlusMatch)
		t.AssertEqualStrings("", readable.String())
	}
}

func TestRingBufferAdvanceToFirstMatchLong(t1 *testing.T) {
	t := ui.T{T: t1}
	var sb strings.Builder

	for i := 0; i < 5000; i += 2 {
		sb.WriteString(" x")
	}

	reader, repool := pool.GetStringReader(sb.String())
	defer repool()
	rb := MakeRingBuffer(reader, 0)
	sut := MakeRingBufferScanner(rb)

	rb.Fill()

	for i := 0; i < 5000; i += 2 {
		readable, offsetPlusMatch, err := sut.FirstMatch(unicorn.Not(unicode.IsSpace))

		if err == ErrBufferEmpty {
			rb.Fill()
			continue
		}

		t.AssertNoError(err)
		rb.AdvanceRead(offsetPlusMatch)
		t.AssertEqualStrings("x", readable.String())
	}
}
