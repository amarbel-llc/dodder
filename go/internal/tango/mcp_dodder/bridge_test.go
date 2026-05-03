package mcp_dodder

import (
	"context"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/lib/charlie/config_cli"
)

func TestBridgeUnknownCommand(t *testing.T) {
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
