package wasm

import (
	"testing"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

func TestWriteStringRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)
	memory := make([]byte, 1024)
	allocator := MakeBumpAllocator(memory, 0)

	input := "hello"
	ptr, length := WriteString(&allocator, input)

	if length != 5 {
		t.Fatalf("expected length 5, got %d", length)
	}

	got := ReadString(memory, ptr, length)
	t.AssertEqualStrings(input, got)
}

func TestWriteStringListRoundTrip(t1 *testing.T) {
	t := ui.MakeT(t1)
	memory := make([]byte, 4096)
	allocator := MakeBumpAllocator(memory, 0)

	input := []string{"alpha", "bravo", "charlie"}
	ptr, count := WriteStringList(&allocator, input)

	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}

	got := ReadStringList(memory, ptr, count)
	t.AssertEqual(input, got)
}
