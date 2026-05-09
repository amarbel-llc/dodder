package catgut

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"unicode"

	"code.linenisgreat.com/dodder/go/lib/alfa/unicorn"
	"github.com/amarbel-llc/purse-first/libs/dewey/alfa/pool"
	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

func TestRingBufferReader(t1 *testing.T) {
	t := ui.MakeT(t1)
	expected := "all that content"
	reader, repool := pool.GetStringReader(expected)
	defer repool()
	sut := MakeRingBuffer(reader, 0)

	var sb strings.Builder

	n, err := io.Copy(&sb, sut)
	t.AssertNoError(err)

	t.AssertEqual(int64(len(expected)), n)

	actual := sb.String()

	t.AssertEqualStrings(expected, actual)
}

func TestRingBufferEmpty(t1 *testing.T) {
	t := ui.MakeT(t1)
	sut := MakeRingBuffer(nil, 10)

	{
		actual := sut.Len()

		t.AssertEqual(0, actual)
	}

	{
		n, err := sut.Write([]byte("test"))

		t.AssertEqual(4, n)

		t.AssertNoError(err)

		{
			expected := 4
			actual := sut.Len()

			t.AssertEqual(expected, actual)
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

		t.AssertEqual(4, n)

		t.AssertEOF(err)

		actual := string(b)

		t.AssertEqualStrings("test", actual)
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
	t := ui.MakeT(t1)
	sut := MakeRingBuffer(nil, 10)

	{
		actual := sut.Len()

		t.AssertEqual(0, actual)
	}

	{
		n, err := sut.Write([]byte("test"))

		t.AssertEqual(4, n)

		t.AssertNoError(err)

		{
			expected := 4
			actual := sut.Len()

			t.AssertEqual(expected, actual)
		}
	}
}

func TestRingBufferEmptyTooBig(t1 *testing.T) {
	t := ui.MakeT(t1)
	sut := MakeRingBuffer(nil, 5)

	for i := 0; i < 11; i++ {
		{
			n, err := sut.Write([]byte("test"))

			t.AssertEqual(4, n)

			t.AssertNoError(err)
		}

		{
			b := make([]byte, 4)
			n, err := sut.Read(b)

			t.AssertEqual(4, n)

			t.AssertEOF(err)

			{
				actual := string(b[:n])
				expected := "test"

				t.AssertEqualStrings(expected, actual)
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
	t := ui.MakeT(t1)
	sut := MakeRingBuffer(bytes.NewBuffer(nil), 3)

	{
		n, err := sut.Write([]byte("teal"))

		t.AssertEqual(3, n)

		t.AssertEOF(err)

		t.AssertEqual(3, sut.Len())
	}

	{
		n, err := sut.Write([]byte("teal"))

		t.AssertEqual(0, n)

		t.AssertEOF(err)

		t.AssertEqual(3, sut.Len())
	}

	{
		b := make([]byte, 4)
		n, err := sut.Read(b)

		{
			expected := 3
			t.AssertEqual(expected, n)

			t.AssertEqual(0, sut.Len())
		}

		t.AssertEOF(err)

		actual := string(b[:n])
		expected := "tea"

		t.AssertEqualStrings(expected, actual)
	}

	{
		b := make([]byte, 4)
		n, err := sut.Read(b)

		{
			expected := 0
			t.AssertEqual(expected, n)
		}

		t.AssertEOF(err)
	}
}

func TestRingBufferDefault(t1 *testing.T) {
	t := ui.MakeT(t1)
	t2 := t.Skip(1)
	sut := MakeRingBuffer(nil, 0)

	one_5 := make([]byte, 2730)
	half := make([]byte, 2048)

	l := 0

	write := func() {
		n, err := sut.Write(one_5)

		t2.AssertEqual(len(one_5), n)

		l += n

		t2.AssertNoError(err)

		t2.AssertEqual(l, sut.Len())
	}

	read := func() {
		n, err := sut.Read(half)

		t2.AssertEqual(len(half), n)

		l -= n

		t2.AssertNoError(err)

		t2.AssertEqual(l, sut.Len())
	}

	write()
	read()
	write()
	read()
	write()
	read()
}

func TestRingBufferDefaultReadFrom(t1 *testing.T) {
	t := ui.MakeT(t1)
	t.SkipTest()

	one_5 := bytes.NewBuffer(make([]byte, 2730))
	sut := MakeRingBuffer(one_5, 0)

	half := make([]byte, 2048)

	l := 0
	t2 := t.Skip(1)

	write := func() {
		n, err := sut.Fill()
		one_5 = bytes.NewBuffer(make([]byte, 2730))

		t2.AssertEqual(one_5.Len(), int(n))

		l += int(n)

		t2.AssertNoError(err)

		t2.AssertEqual(l, sut.Len())
	}

	read := func() {
		n, err := sut.Read(half)

		t2.AssertEqual(len(half), n)

		l -= n

		t2.AssertNoError(err)

		t2.AssertEqual(l, sut.Len())
	}

	write()
	read()
	write()
	read()
	write()
	read()
}

func TestRingBufferPeekUpto2(t1 *testing.T) {
	t := ui.MakeT(t1)
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
	t := ui.MakeT(t1)
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
	t := ui.MakeT(t1)
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
