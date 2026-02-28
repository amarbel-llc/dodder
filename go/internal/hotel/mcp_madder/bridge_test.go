package mcp_madder

import (
	"context"
	"testing"

	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/lib/foxtrot/config_cli"
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
