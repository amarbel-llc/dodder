package mcp_dodder

import (
	"context"
	"sync"

	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

type BridgeResult struct {
	Stdout    string
	Stderr    string
	Truncated bool
	BytesSeen int
}

type Bridge struct {
	utility command.Utility

	// mu serializes RunCommand. The go-mcp server handles every
	// incoming message on its own goroutine, but the commands the
	// bridge runs are CLI-shaped: the registry hands back long-lived
	// command values whose flag-bound state is shared across
	// invocations, so reset/parse/run must not interleave (#247).
	// Pointer so Bridge copies (it is passed by value) share the lock.
	mu *sync.Mutex
}

func MakeBridge(utility command.Utility) Bridge {
	return Bridge{
		utility: utility,
		mu:      &sync.Mutex{},
	}
}

func (b Bridge) RunCommand(
	ctx context.Context,
	cmdName string,
	cliArgs []string,
	maxBytes int,
) (BridgeResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	outWriter := MakeLimitingWriter(maxBytes)
	errWriter := MakeLimitingWriter(maxBytes)

	config := repo_config_cli.Default()
	config.CustomOut = outWriter
	config.CustomErr = errWriter

	utility := command.MakeUtility("dodder", config)

	for name, cmd := range b.utility.AllCmds() {
		utility.AddCmd(name, cmd)
	}

	// The registry hands back long-lived command values shared across
	// every bridge invocation (and with the CLI registration in the
	// same process). Commands with accumulating flag-bound state
	// implement ResetCLIState so each tool call parses flags against a
	// clean slate — without this, two `new` calls concatenate
	// descriptions (#247).
	if cmd, ok := b.utility.GetCmd(cmdName); ok {
		if resetter, ok := cmd.(command.CommandWithResetCLIState); ok {
			resetter.ResetCLIState()
		}
	}

	errCtx := errors.MakeContext(ctx)

	args := make([]string, 0, 2+len(cliArgs))
	args = append(args, "dodder", cmdName)
	args = append(args, cliArgs...)

	var result BridgeResult

	if err := errCtx.Run(func(ctx errors.Context) {
		cmd, flagSet, ok := utility.MakeCmdAndFlagSet(ctx, args)
		if !ok {
			return
		}

		req, ok := utility.MakeRequest(ctx, cmd, flagSet)
		if !ok {
			return
		}

		cmd.Run(req)
	}); err != nil {
		return result, err
	}

	result = BridgeResult{
		Stdout:    outWriter.String(),
		Stderr:    errWriter.String(),
		Truncated: outWriter.Truncated(),
		BytesSeen: outWriter.BytesSeen(),
	}

	return result, nil
}
