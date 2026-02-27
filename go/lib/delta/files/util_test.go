package files

import (
	"reflect"
	"testing"

	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

func TestPathElements(t1 *testing.T) {
	t := ui.T{T: t1}

	path := "/wow/ok/great.ext"
	expected := []string{"ext", "great", "ok", "wow"}
	actual := PathElements(path)

	if reflect.DeepEqual(expected, actual) {
		t.AssertEqual(expected, actual)
	}
}
