package format

import (
	"strings"
	"testing"
)

func TestWriter1(t *testing.T) {
	w := NewLineWriter()

	w.WriteLines(
		"one",
		"two",
	)

	w.WriteFormats("%s", "three")

	sb := &strings.Builder{}
	expected := `one
two
three
`

	n, err := w.WriteTo(sb)

	if n != int64(len(expected)) {
		t.Fatalf("expected length %d but got %d", len(expected), n)
	}

	if err != nil {
		t.Fatalf("%s", err)
	}

	if sb.String() != expected {
		t.Fatalf("\n expected: %q\n    actual: %q", expected, sb.String())
	}

	return
}
