package commands_dodder

import (
	"os"

	"code.linenisgreat.com/dodder/go/internal/bravo/claude_hooks"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
)

func init() {
	utility.AddCmd("hook", &Hook{})
}

// Hook implements the Claude Code hook protocol for the dodder clown
// plugin: the plugin's hooks/handler script execs `dodder hook` on
// every PreToolUse event, and claude_hooks.Run decides which dodder
// MCP tools are auto-approved (#244). Reads one hook event from stdin,
// writes a permission decision (or nothing) to stdout.
type Hook struct{}

func (cmd Hook) GetDescription() command.Description {
	return command.Description{
		Short: "respond to a Claude Code hook event from the clown plugin",
	}
}

func (cmd Hook) Run(req command.Request) {
	if err := claude_hooks.Run(os.Stdin, os.Stdout); err != nil {
		req.Cancel(err)
	}
}
