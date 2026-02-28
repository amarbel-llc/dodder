package wasm

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestWriteStringRoundTrip(t *testing.T) {
	memory := make([]byte, 1024)
	allocator := MakeBumpAllocator(memory, 0)

	input := "hello"
	ptr, length := WriteString(&allocator, input)

	if length != 5 {
		t.Fatalf("expected length 5, got %d", length)
	}

	got := ReadString(memory, ptr, length)
	if got != input {
		t.Fatalf("expected %q, got %q", input, got)
	}
}

func TestWriteStringListRoundTrip(t *testing.T) {
	memory := make([]byte, 4096)
	allocator := MakeBumpAllocator(memory, 0)

	input := []string{"alpha", "bravo", "charlie"}
	ptr, count := WriteStringList(&allocator, input)

	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}

	got := ReadStringList(memory, ptr, count)
	if diff := cmp.Diff(input, got); diff != "" {
		t.Fatalf("mismatch (-want +got):\n%s", diff)
	}
}
