package catgut

import (
	"io"
	"strings"
	"testing"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

func TestSliceReader(t1 *testing.T) {
	t := ui.MakeT(t1)
	input := Slice{
		data: [2][]byte{
			[]byte("test"),
			[]byte("string"),
		},
	}

	sut := MakeSliceReader(input)

	var actual strings.Builder

	n1, err := io.Copy(&actual, sut)
	t.AssertNoError(err)
	n := int(n1)

	t.AssertEqual(input.Len(), n)
}
