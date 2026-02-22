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

	tools.Register(
		"madder_cat",
		"Output blob contents by SHA digest",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"sha": {"type": "string", "description": "SHA digest of the blob to read"},
				"prefix_sha": {"type": "boolean", "description": "Prefix each line with the SHA digest"},
				"blob_store": {"type": "integer", "description": "Blob store index to read from"}
			},
			"required": ["sha"],
			"additionalProperties": false
		}`),
		makeBridgeHandler(bridge, "cat", func(args json.RawMessage) ([]string, error) {
			var p struct {
				SHA       string `json:"sha"`
				PrefixSha bool   `json:"prefix_sha"`
				BlobStore *int   `json:"blob_store"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			var cliArgs []string
			if p.PrefixSha {
				cliArgs = append(cliArgs, "-prefix-sha")
			}
			if p.BlobStore != nil {
				cliArgs = append(cliArgs, "-blob-store", fmt.Sprintf("%d", *p.BlobStore))
			}
			cliArgs = append(cliArgs, p.SHA)
			return cliArgs, nil
		}),
	)

	tools.Register(
		"madder_cat_ids",
		"List all blob IDs in one or more blob stores",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"blob_store_ids": {
					"type": "array",
					"items": {"type": "string"},
					"description": "Blob store IDs to list from (defaults to all stores if omitted)"
				}
			},
			"additionalProperties": false
		}`),
		makeBridgeHandler(bridge, "cat-ids", func(args json.RawMessage) ([]string, error) {
			var p struct {
				BlobStoreIds []string `json:"blob_store_ids"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			return p.BlobStoreIds, nil
		}),
	)

	tools.Register(
		"madder_info_repo",
		"Query blob store configuration and repository info",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"blob_store_index": {
					"type": "string",
					"description": "Blob store index to query (optional, defaults to the default blob store)"
				},
				"keys": {
					"type": "array",
					"items": {"type": "string"},
					"description": "Config keys to query (e.g. config-immutable, compression-type, xdg). Defaults to config-immutable if omitted."
				}
			},
			"additionalProperties": false
		}`),
		makeBridgeHandler(bridge, "info-repo", func(args json.RawMessage) ([]string, error) {
			var p struct {
				BlobStoreIndex string   `json:"blob_store_index"`
				Keys           []string `json:"keys"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			var cliArgs []string
			if p.BlobStoreIndex != "" {
				cliArgs = append(cliArgs, p.BlobStoreIndex)
				cliArgs = append(cliArgs, p.Keys...)
			} else if len(p.Keys) == 1 {
				cliArgs = append(cliArgs, p.Keys[0])
			} else if len(p.Keys) > 1 {
				// Without a blob store index, only a single key is supported
				// as a positional arg. Multiple keys require the blob store index.
				cliArgs = append(cliArgs, p.Keys[0])
			}
			return cliArgs, nil
		}),
	)

	tools.Register(
		"madder_fsck",
		"Check blob store integrity by verifying all blobs",
		json.RawMessage(`{
			"type": "object",
			"properties": {
				"blob_store_ids": {
					"type": "array",
					"items": {"type": "string"},
					"description": "Blob store IDs to check (defaults to all stores if omitted)"
				}
			},
			"additionalProperties": false
		}`),
		makeBridgeHandler(bridge, "fsck", func(args json.RawMessage) ([]string, error) {
			var p struct {
				BlobStoreIds []string `json:"blob_store_ids"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			return p.BlobStoreIds, nil
		}),
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
