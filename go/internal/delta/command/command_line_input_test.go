package command

import (
	"testing"

	"code.linenisgreat.com/dodder/go/lib/0/collections_slice"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

// Regression for #182: LastCompleteArg decremented argc for an in-progress
// token but then returned .Last() (the in-progress token itself) instead of
// the element just before it.
func TestCommandLineInputLastCompleteArgStripsInProgress(t1 *testing.T) {
	t := ui.MakeT(t1)

	cli := CommandLineInput{
		FlagsOrArgs: collections_slice.Make("a", "b", "in-prog"),
		InProgress:  "in-prog",
	}

	arg, ok := cli.LastCompleteArg()
	if !ok {
		t.Errorf("expected ok=true")
	}

	t.AssertEqualStrings("b", arg)
}

func TestCommandLineInputLastCompleteArgNoInProgress(t1 *testing.T) {
	t := ui.MakeT(t1)

	cli := CommandLineInput{
		FlagsOrArgs: collections_slice.Make("a", "b"),
	}

	arg, ok := cli.LastCompleteArg()
	if !ok {
		t.Errorf("expected ok=true")
	}

	t.AssertEqualStrings("b", arg)
}

func TestCommandLineInputLastCompleteArgOnlyInProgress(t1 *testing.T) {
	t := ui.MakeT(t1)

	cli := CommandLineInput{
		FlagsOrArgs: collections_slice.Make("in-prog"),
		InProgress:  "in-prog",
	}

	arg, ok := cli.LastCompleteArg()
	if ok {
		t.Errorf("expected ok=false when only the in-progress token is present, got %q", arg)
	}

	t.AssertEqualStrings("", arg)
}
