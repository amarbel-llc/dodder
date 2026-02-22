package mcp_madder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"code.linenisgreat.com/dodder/go/src/juliett/command"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/protocol"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/server"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/transport"
)

const defaultMaxBytes = 100_000

func RunServer(utility command.Utility) error {
	bridge := MakeBridge(utility)
	tools := server.NewToolRegistry()

	registerTools(tools, bridge)

	t := transport.NewStdio(os.Stdin, os.Stdout)
	srv, err := server.New(t, server.Options{
		ServerName:    "madder",
		ServerVersion: "0.1.0",
		Tools:         tools,
	})
	if err != nil {
		return err
	}

	return srv.Run(context.Background())
}

func registerTools(tools *server.ToolRegistry, bridge Bridge) {
	tools.Register(
		"madder_list",
		"List available blob stores with their IDs and descriptions",
		json.RawMessage(`{
			"type": "object",
			"properties": {},
			"additionalProperties": false
		}`),
		makeBridgeHandler(bridge, "list", nil),
	)
}

type paramTranslator func(args json.RawMessage) ([]string, error)

func makeBridgeHandler(
	bridge Bridge,
	cmdName string,
	translate paramTranslator,
) server.ToolHandler {
	return func(
		ctx context.Context,
		args json.RawMessage,
	) (*protocol.ToolCallResult, error) {
		var cliArgs []string

		if translate != nil {
			var err error
			if cliArgs, err = translate(args); err != nil {
				return protocol.ErrorResult(
					fmt.Sprintf("Invalid arguments: %v", err),
				), nil
			}
		}

		result, err := bridge.RunCommand(ctx, cmdName, cliArgs, defaultMaxBytes)
		if err != nil {
			return protocol.ErrorResult(err.Error()), nil
		}

		output := result.Stdout
		if result.Truncated {
			output += fmt.Sprintf(
				"\n\n[truncated: showed %d of %d bytes]",
				len(result.Stdout),
				result.BytesSeen,
			)
		}

		if result.Stderr != "" {
			output += "\n\nstderr:\n" + result.Stderr
		}

		return &protocol.ToolCallResult{
			Content: []protocol.ContentBlock{
				protocol.TextContent(output),
			},
		}, nil
	}
}
