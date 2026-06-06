package mcp_dodder

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

// TestNewToolCLIArgsForcesNonInteractive pins the #233 fix: every
// `new` tool invocation must pass -edit=false so the bridge never
// launches an editor in the no-TTY MCP context (which would hang the
// tool call). The flag is required regardless of which optional fields
// the caller supplies.
func TestNewToolCLIArgsForcesNonInteractive(t1 *testing.T) {
	t := ui.MakeT(t1)

	cases := []string{
		`{}`,
		`{"description":"a note"}`,
		`{"description":"a note","type":"!md","tags":["x","y"]}`,
	}

	for _, in := range cases {
		cliArgs, err := newToolCLIArgs(json.RawMessage(in))
		t.AssertNoError(err)

		if !slices.Contains(cliArgs, "-edit=false") {
			t.Fatalf(
				"newToolCLIArgs(%s) = %v; missing -edit=false (would launch an editor and hang)",
				in, cliArgs,
			)
		}
	}
}
