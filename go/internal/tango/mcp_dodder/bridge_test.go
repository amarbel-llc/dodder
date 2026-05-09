package mcp_dodder

import (
	"context"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/lib/charlie/config_cli"
	"github.com/amarbel-llc/purse-first/libs/dewey/charlie/ui"
)

func TestBridgeUnknownCommand(t1 *testing.T) {
	t := ui.MakeT(t1)
	utility := command.MakeUtility("dodder", config_cli.Default())
	bridge := MakeBridge(utility)
	_, err := bridge.RunCommand(
		context.Background(),
		"nonexistent-command",
		nil,
		100_000,
	)
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}
