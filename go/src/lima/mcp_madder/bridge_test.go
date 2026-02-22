package mcp_madder

import (
	"context"
	"testing"
)

func TestBridgeUnknownCommand(t *testing.T) {
	bridge := MakeBridge()
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
