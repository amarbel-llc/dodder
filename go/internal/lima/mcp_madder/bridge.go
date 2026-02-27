package mcp_madder

import (
	"context"

	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/echo/config_cli"
	"code.linenisgreat.com/dodder/go/internal/juliett/command"
)

type BridgeResult struct {
	Stdout    string
	Stderr    string
	Truncated bool
	BytesSeen int
}

type Bridge struct {
	utility command.Utility
}

func MakeBridge(utility command.Utility) Bridge {
	return Bridge{
		utility: utility,
	}
}

func (b Bridge) RunCommand(
	ctx context.Context,
	cmdName string,
	cliArgs []string,
	maxBytes int,
) (BridgeResult, error) {
	outWriter := MakeLimitingWriter(maxBytes)
	errWriter := MakeLimitingWriter(maxBytes)

	config := &config_cli.Config{
		CustomOut: outWriter,
		CustomErr: errWriter,
	}

	utility := command.MakeUtility("madder", config)

	for name, cmd := range b.utility.AllCmds() {
		utility.AddCmd(name, cmd)
	}

	errCtx := errors.MakeContext(ctx)

	args := make([]string, 0, 2+len(cliArgs))
	args = append(args, "madder", cmdName)
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
