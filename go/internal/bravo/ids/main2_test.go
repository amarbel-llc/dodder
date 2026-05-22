package ids

import (
	"bytes"
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

func idWriteToReadFromData() []string {
	return []string{
		"one/uno",
		"config",
		"!md",
		"-tag",
		"/repo",
		"tag",
	}
}

func TestIdWriteToReadFrom(t1 *testing.T) {
	t := ui.MakeT(t1)

	for _, value := range idWriteToReadFromData() {
		var id ObjectId
		t.AssertNoError(id.Set(value))

		var b bytes.Buffer

		_, err := id.WriteTo(&b)
		t.AssertNoError(err)

		var id2 ObjectId

		_, err = id2.ReadFrom(&b)
		t.AssertNoError(err)

		t.AssertEqual(id.String(), id2.String())
	}
}
