//go:build test

package mcp_dodder

import (
	"context"
	"slices"
	"sync"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/madder/go/pkgs/scoped_id"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

// appendingValue mimics descriptions.Description's appending Set: a
// second Set joins instead of replacing. Bound via flagSet.Var, it
// never self-heals from a registered default the way scalar flags do.
type appendingValue struct {
	value string
}

func (v *appendingValue) Set(s string) error {
	if v.value != "" {
		v.value = v.value + " " + s
	} else {
		v.value = s
	}

	return nil
}

func (v *appendingValue) String() string {
	return v.value
}

// leakyCmd stands in for commands like `new` whose registered
// singleton carries accumulating flag-bound state between bridge
// invocations (#247).
type leakyCmd struct {
	description appendingValue

	mu   sync.Mutex
	seen []string
}

func (cmd *leakyCmd) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.Var(&cmd.description, "description", "")
}

func (cmd *leakyCmd) ResetCLIState() {
	cmd.description = appendingValue{}
}

func (cmd *leakyCmd) Run(req command.Request) {
	cmd.mu.Lock()
	defer cmd.mu.Unlock()
	cmd.seen = append(cmd.seen, cmd.description.value)
}

var (
	_ interfaces.CommandComponentWriter = (*leakyCmd)(nil)
	_ command.CommandWithResetCLIState  = (*leakyCmd)(nil)
)

// TestBridgeResetsCLIStateBetweenInvocations pins the #247 fix: the
// bridge reuses the registered command values across tool calls, so
// it must call ResetCLIState before each invocation. Without the
// reset, the second run sees "first second".
func TestBridgeResetsCLIStateBetweenInvocations(t1 *testing.T) {
	t := ui.MakeT(t1)

	utility := command.MakeUtility("dodder", repo_config_cli.Default())
	cmd := &leakyCmd{}
	utility.AddCmd("leaky", cmd)

	bridge := MakeBridge(utility, scoped_id.Id{}, false)

	for _, description := range []string{"first", "second"} {
		if _, err := bridge.RunCommand(
			context.Background(),
			"leaky",
			[]string{"-description", description},
			1<<16,
		); err != nil {
			t.Fatalf("RunCommand(%q): %s", description, err)
		}
	}

	expected := []string{"first", "second"}

	if !slices.Equal(cmd.seen, expected) {
		t.Fatalf(
			"command saw descriptions %q, expected %q (state leaked between bridge invocations)",
			cmd.seen,
			expected,
		)
	}
}

// TestBridgeSerializesConcurrentInvocations pins the second half of
// the #247 fix: the go-mcp server handles each message on its own
// goroutine, so without the bridge mutex two tool calls interleave
// reset/parse/run on the shared command value and one call runs with
// the other's flag state. Without the mutex this test is flaky rather
// than deterministically red (it is a race); the race-detector lane
// catches it reliably.
func TestBridgeSerializesConcurrentInvocations(t1 *testing.T) {
	t := ui.MakeT(t1)

	utility := command.MakeUtility("dodder", repo_config_cli.Default())
	cmd := &leakyCmd{}
	utility.AddCmd("leaky", cmd)

	bridge := MakeBridge(utility, scoped_id.Id{}, false)

	descriptions := []string{"one", "two", "three", "four", "five", "six"}

	var wg sync.WaitGroup

	for _, description := range descriptions {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if _, err := bridge.RunCommand(
				context.Background(),
				"leaky",
				[]string{"-description", description},
				1<<16,
			); err != nil {
				t.Errorf("RunCommand(%q): %s", description, err)
			}
		}()
	}

	wg.Wait()

	seen := slices.Clone(cmd.seen)
	slices.Sort(seen)

	expected := slices.Clone(descriptions)
	slices.Sort(expected)

	if !slices.Equal(seen, expected) {
		t.Fatalf(
			"concurrent invocations saw %q, expected a permutation of %q",
			cmd.seen,
			descriptions,
		)
	}
}
