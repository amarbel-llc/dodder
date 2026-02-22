package mcp_madder

import (
	"context"
	"testing"

	"code.linenisgreat.com/dodder/go/src/echo/config_cli"
	"code.linenisgreat.com/dodder/go/src/juliett/command"
)

func TestBridgeUnknownCommand(t *testing.T) {
	utility := command.MakeUtility("madder", config_cli.Default())
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
