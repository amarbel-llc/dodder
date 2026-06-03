package command

import (
	"testing"

	"code.linenisgreat.com/dodder/go/lib/0/collections_slice"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

// Regression for #183: LastArg used to evaluate PopArgs() (which advances Argi
// to the end) before RemainingArgCount()-1, leaving the index at -1 and
// panicking. Mirrors amarbel-llc/cutting-garden's TestRequest_LastArg_*.
func TestRequestLastArg(t1 *testing.T) {
	t := ui.MakeT(t1)

	req := Request{
		input: &CommandLineInput{
			Args: collections_slice.Make("a", "b", "c"),
		},
	}

	arg, ok := req.LastArg()
	if !ok {
		t.Errorf("expected ok=true for non-empty args")
	}

	t.AssertEqualStrings("c", arg)
}

func TestRequestLastArgEmpty(t1 *testing.T) {
	t := ui.MakeT(t1)

	req := Request{input: &CommandLineInput{}}

	arg, ok := req.LastArg()
	if ok {
		t.Errorf("expected ok=false for empty args, got arg=%q", arg)
	}

	t.AssertEqualStrings("", arg)
}
