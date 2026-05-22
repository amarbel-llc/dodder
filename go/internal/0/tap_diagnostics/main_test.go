package tap_diagnostics_test

import (
	"fmt"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/0/tap_diagnostics"
	"github.com/amarbel-llc/madder/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

func TestFromErrNotEqual(t1 *testing.T) {
	t := ui.MakeT(t1)
	var expected, actual markl.Id

	err := markl.ErrNotEqual{
		Expected: &expected,
		Actual:   &actual,
	}

	diag := tap_diagnostics.FromError(err)

	t.AssertEqualStrings("fail", diag["severity"])
	_, ok := diag["expected"]
	t.AssertTrue(ok, "expected 'expected' field to be set")
	_, ok = diag["actual"]
	t.AssertTrue(ok, "expected 'actual' field to be set")
}

func TestFromErrIsNull(t1 *testing.T) {
	t := ui.MakeT(t1)
	err := markl.ErrIsNull{Purpose: "object-dig"}

	diag := tap_diagnostics.FromError(err)

	t.AssertEqualStrings("fail", diag["severity"])
	t.AssertEqualStrings("object-dig", diag["field"])
}

func TestFromGenericError(t1 *testing.T) {
	t := ui.MakeT(t1)
	err := fmt.Errorf("something went wrong")

	diag := tap_diagnostics.FromError(err)

	t.AssertEqualStrings("fail", diag["severity"])
	t.AssertEqualStrings("something went wrong", diag["message"])
}
