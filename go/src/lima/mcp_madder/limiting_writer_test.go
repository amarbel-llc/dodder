package mcp_madder

import (
	"strings"
	"testing"
)

func TestLimitingWriterUnderLimit(t *testing.T) {
	w := MakeLimitingWriter(100)
	n, err := w.Write([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Fatalf("expected n=5, got %d", n)
	}
	if w.String() != "hello" {
		t.Fatalf("expected 'hello', got %q", w.String())
	}
	if w.Truncated() {
		t.Fatal("should not be truncated")
	}
}

func TestLimitingWriterOverLimit(t *testing.T) {
	w := MakeLimitingWriter(10)
	data := strings.Repeat("x", 20)
	n, err := w.Write([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if n != 20 {
		t.Fatalf("expected n=20, got %d", n)
	}
	if len(w.String()) > 10 {
		t.Fatalf("buffer should be at most 10 bytes, got %d", len(w.String()))
	}
	if !w.Truncated() {
		t.Fatal("should be truncated")
	}
	if w.BytesSeen() != 20 {
		t.Fatalf("expected 20 bytes seen, got %d", w.BytesSeen())
	}
}

func TestLimitingWriterMultipleWrites(t *testing.T) {
	w := MakeLimitingWriter(10)
	w.Write([]byte("12345"))
	w.Write([]byte("67890"))
	w.Write([]byte("overflow"))
	if w.String() != "1234567890" {
		t.Fatalf("expected '1234567890', got %q", w.String())
	}
	if !w.Truncated() {
		t.Fatal("should be truncated")
	}
	if w.BytesSeen() != 18 {
		t.Fatalf("expected 18 bytes seen, got %d", w.BytesSeen())
	}
}

func TestLimitingWriterStringWriter(t *testing.T) {
	w := MakeLimitingWriter(100)
	n, err := w.WriteString("hello")
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Fatalf("expected n=5, got %d", n)
	}
	if w.String() != "hello" {
		t.Fatalf("expected 'hello', got %q", w.String())
	}
}
