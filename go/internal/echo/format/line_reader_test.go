package format

import (
	"fmt"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/alfa/pool"
	"code.linenisgreat.com/dodder/go/internal/bravo/ohio"
	"code.linenisgreat.com/dodder/go/internal/bravo/ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/values"
)

func TestLineReaderOneReaderHappy(t1 *testing.T) {
	t := ui.T{T: t1}

	input := "test string\n"
	test_value := values.MakeString("test string")
	r, repool := pool.GetStringReader(input)
	defer repool()
	sut := MakeLineReaderConsumeEmpty(
		test_value.Match,
	)

	n, err := sut.ReadFrom(r)
	if err != nil {
		t.Fatalf("expected no error but got %s", err)
	}

	exN := int64(12)
	if n != exN {
		t.Fatalf("expected to read %d but read %d", exN, n)
	}
}

func TestLineReaderOneReaderSad(t1 *testing.T) {
	t := ui.T{T: t1}

	input := "test string sad\n"
	test_value := values.MakeString("test string")
	r, repool := pool.GetStringReader(input)
	defer repool()
	sut := MakeLineReaderConsumeEmpty(
		test_value.Match,
	)

	n, err := sut.ReadFrom(r)
	if err == nil {
		t.Fatalf("expected an error but got none")
	}

	exN := int64(16)
	if n != exN {
		t.Fatalf("expected to read %d but read %d", exN, n)
	}
}

func TestLineReaderTwoReaders(t1 *testing.T) {
	t := ui.T{T: t1}

	test_value_one := values.MakeString("test string")
	test_value_two := values.MakeString("test string two")

	input := fmt.Sprintf("%s\n%s\n", test_value_one, test_value_two)

	r, repool := pool.GetStringReader(input)
	defer repool()
	sut := MakeLineReaderConsumeEmpty(
		ohio.MakeLineReaderIterateStrict(
			test_value_one.Match,
			test_value_two.Match,
		),
	)

	n, err := sut.ReadFrom(r)
	if err != nil {
		t.Fatalf("expected no error but got %s", err)
	}

	exN := int64(28)
	if n != exN {
		t.Fatalf("expected to read %d but read %d", exN, n)
	}
}

func TestLineReaderTwoReadersSad(t1 *testing.T) {
	t := ui.T{T: t1}

	test_value_one := values.MakeString("test string")
	test_value_two := values.MakeString("test string two")

	input := fmt.Sprintf("%s\n%s sad\n", test_value_one, test_value_two)

	r, repool := pool.GetStringReader(input)
	defer repool()
	sut := MakeLineReaderConsumeEmpty(
		ohio.MakeLineReaderIterateStrict(
			test_value_one.Match,
			test_value_two.Match,
		),
	)

	n, err := sut.ReadFrom(r)
	if err == nil {
		t.Fatalf("expected an error but got none")
	}

	exN := int64(32)
	if n != exN {
		t.Fatalf("expected to read %d but read %d", exN, n)
	}
}
