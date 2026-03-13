package mcp_madder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/protocol"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/server"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/transport"
)

const defaultMaxBytes = 100_000

var (
	readOnlyAnnotations = &protocol.ToolAnnotations{
		ReadOnlyHint:   protocol.BoolPtr(true),
		IdempotentHint: protocol.BoolPtr(true),
	}

	writeAnnotations = &protocol.ToolAnnotations{
		ReadOnlyHint:    protocol.BoolPtr(false),
		DestructiveHint: protocol.BoolPtr(false),
	}

	destructiveAnnotations = &protocol.ToolAnnotations{
		ReadOnlyHint:    protocol.BoolPtr(false),
		DestructiveHint: protocol.BoolPtr(true),
	}
)

func RunServer(utility command.Utility) error {
	bridge := MakeBridge(utility)
	tools := server.NewToolRegistryV1()

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

func registerTools(tools *server.ToolRegistryV1, bridge Bridge) {
	tools.Register(
		protocol.ToolV1{
			Name:        "madder_list",
			Description: "List available blob stores with their IDs and descriptions",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {},
				"additionalProperties": false
			}`),
			Annotations: readOnlyAnnotations,
		},
		makeBridgeHandler(bridge, "list", nil),
	)

	tools.Register(
		protocol.ToolV1{
			Name:        "madder_cat",
			Description: "Output blob contents by SHA digest",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"sha": {"type": "string", "description": "SHA digest of the blob to read"},
					"prefix_sha": {"type": "boolean", "description": "Prefix each line with the SHA digest"},
					"blob_store": {"type": "integer", "description": "Blob store index to read from"}
				},
				"required": ["sha"],
				"additionalProperties": false
			}`),
			Annotations: readOnlyAnnotations,
		},
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
		protocol.ToolV1{
			Name:        "madder_cat_ids",
			Description: "List all blob IDs in one or more blob stores",
			InputSchema: json.RawMessage(`{
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
			Annotations: readOnlyAnnotations,
		},
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
		protocol.ToolV1{
			Name:        "madder_info_repo",
			Description: "Query blob store configuration and repository info",
			InputSchema: json.RawMessage(`{
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
			Annotations: readOnlyAnnotations,
		},
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
		protocol.ToolV1{
			Name:        "madder_fsck",
			Description: "Check blob store integrity by verifying all blobs",
			InputSchema: json.RawMessage(`{
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
			Annotations: readOnlyAnnotations,
		},
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

	tools.Register(
		protocol.ToolV1{
			Name:        "madder_write",
			Description: "Write files into the blob store. Paths are file paths or '-' for stdin. Can also accept a blob store ID to target a specific store.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"paths": {
						"type": "array",
						"items": {"type": "string"},
						"description": "File paths to write into the blob store"
					},
					"check": {
						"type": "boolean",
						"description": "Only check if the object already exists without writing"
					}
				},
				"required": ["paths"],
				"additionalProperties": false
			}`),
			Annotations: writeAnnotations,
		},
		makeBridgeHandler(bridge, "write", func(args json.RawMessage) ([]string, error) {
			var p struct {
				Paths []string `json:"paths"`
				Check bool     `json:"check"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			var out []string
			if p.Check {
				out = append(out, "-check")
			}
			out = append(out, p.Paths...)
			return out, nil
		}),
	)

	// NOTE: read consumes JSON from stdin ({"blob": "..."}). The bridge does
	// not currently support stdin piping, so this tool has limited utility
	// until stdin support is added. The "input" parameter is accepted but
	// cannot be delivered to the command's stdin.
	tools.Register(
		protocol.ToolV1{
			Name:        "madder_read",
			Description: "Read blob content from JSON input. Each JSON object should have a 'blob' field with the content to store. Known limitation: stdin piping is not yet supported in the MCP bridge.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"input": {
						"type": "string",
						"description": "JSON input with blob entries, e.g. {\"blob\": \"content\"}. NOTE: stdin piping not yet supported in MCP bridge"
					}
				},
				"additionalProperties": false
			}`),
			Annotations: writeAnnotations,
		},
		makeBridgeHandler(bridge, "read", nil),
	)

	tools.Register(
		protocol.ToolV1{
			Name:        "madder_sync",
			Description: "Sync blobs between stores. With no args, syncs default store to all others. With args, first is source store ID, rest are destination store IDs.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"source": {
						"type": "string",
						"description": "Source blob store ID (omit to use default)"
					},
					"destinations": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Destination blob store IDs (omit to sync to all non-source stores)"
					},
					"limit": {
						"type": "integer",
						"description": "Stop after syncing this many blobs (0 = no limit)"
					}
				},
				"additionalProperties": false
			}`),
			Annotations: writeAnnotations,
		},
		makeBridgeHandler(bridge, "sync", func(args json.RawMessage) ([]string, error) {
			var p struct {
				Source       string   `json:"source"`
				Destinations []string `json:"destinations"`
				Limit        int      `json:"limit"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			var out []string
			if p.Limit > 0 {
				out = append(out, "-limit", fmt.Sprintf("%d", p.Limit))
			}
			if p.Source != "" {
				out = append(out, p.Source)
			}
			out = append(out, p.Destinations...)
			return out, nil
		}),
	)

	tools.Register(
		protocol.ToolV1{
			Name:        "madder_init",
			Description: "Initialize a new default blob store with the given ID",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"blob_store_id": {
						"type": "string",
						"description": "Identifier for the new blob store"
					}
				},
				"required": ["blob_store_id"],
				"additionalProperties": false
			}`),
			Annotations: destructiveAnnotations,
		},
		makeBridgeHandler(bridge, "init", func(args json.RawMessage) ([]string, error) {
			var p struct {
				BlobStoreId string `json:"blob_store_id"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			return []string{p.BlobStoreId}, nil
		}),
	)

	tools.Register(
		protocol.ToolV1{
			Name:        "madder_init_from",
			Description: "Initialize a blob store from an existing config file. Reads the config, upgrades it if needed, and creates a new store.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"blob_store_id": {
						"type": "string",
						"description": "Identifier for the new blob store"
					},
					"config_path": {
						"type": "string",
						"description": "Path to an existing blob store config file"
					}
				},
				"required": ["blob_store_id", "config_path"],
				"additionalProperties": false
			}`),
			Annotations: destructiveAnnotations,
		},
		makeBridgeHandler(bridge, "init-from", func(args json.RawMessage) ([]string, error) {
			var p struct {
				BlobStoreId string `json:"blob_store_id"`
				ConfigPath  string `json:"config_path"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			return []string{p.BlobStoreId, p.ConfigPath}, nil
		}),
	)

	tools.Register(
		protocol.ToolV1{
			Name:        "madder_init_inventory_archive",
			Description: "Initialize an inventory archive blob store with delta compression support",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"blob_store_id": {
						"type": "string",
						"description": "Identifier for the new inventory archive blob store"
					}
				},
				"required": ["blob_store_id"],
				"additionalProperties": false
			}`),
			Annotations: destructiveAnnotations,
		},
		makeBridgeHandler(bridge, "init-inventory-archive", func(args json.RawMessage) ([]string, error) {
			var p struct {
				BlobStoreId string `json:"blob_store_id"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			return []string{p.BlobStoreId}, nil
		}),
	)

	tools.Register(
		protocol.ToolV1{
			Name:        "madder_init_pointer",
			Description: "Initialize a pointer blob store that references another blob store",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"blob_store_id": {
						"type": "string",
						"description": "Identifier for the new pointer blob store"
					},
					"id": {
						"type": "string",
						"description": "ID of the blob store to point to"
					},
					"base_path": {
						"type": "string",
						"description": "Path to the referenced blob store base directory"
					},
					"config_path": {
						"type": "string",
						"description": "Path to the referenced blob store config file"
					}
				},
				"required": ["blob_store_id"],
				"additionalProperties": false
			}`),
			Annotations: destructiveAnnotations,
		},
		makeBridgeHandler(bridge, "init-pointer", func(args json.RawMessage) ([]string, error) {
			var p struct {
				BlobStoreId string `json:"blob_store_id"`
				Id          string `json:"id"`
				BasePath    string `json:"base_path"`
				ConfigPath  string `json:"config_path"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			var out []string
			if p.Id != "" {
				out = append(out, "-id", p.Id)
			}
			if p.BasePath != "" {
				out = append(out, "-base-path", p.BasePath)
			}
			if p.ConfigPath != "" {
				out = append(out, "-config-path", p.ConfigPath)
			}
			out = append(out, p.BlobStoreId)
			return out, nil
		}),
	)

	tools.Register(
		protocol.ToolV1{
			Name:        "madder_pack",
			Description: "Pack loose blobs into archives for inventory archive blob stores",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"store": {
						"type": "string",
						"description": "Inventory archive store ID to pack (omit to pack all)"
					},
					"delete_loose": {
						"type": "boolean",
						"description": "Validate archive then delete packed loose blobs"
					}
				},
				"additionalProperties": false
			}`),
			Annotations: writeAnnotations,
		},
		makeBridgeHandler(bridge, "pack", func(args json.RawMessage) ([]string, error) {
			var p struct {
				Store       string `json:"store"`
				DeleteLoose bool   `json:"delete_loose"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			var out []string
			if p.Store != "" {
				out = append(out, "-store", p.Store)
			}
			if p.DeleteLoose {
				out = append(out, "-delete-loose")
			}
			return out, nil
		}),
	)
}

type paramTranslator func(args json.RawMessage) ([]string, error)

func makeBridgeHandler(
	bridge Bridge,
	cmdName string,
	translate paramTranslator,
) server.ToolHandlerV1 {
	return func(
		ctx context.Context,
		args json.RawMessage,
	) (*protocol.ToolCallResultV1, error) {
		var cliArgs []string

		if translate != nil {
			var err error
			if cliArgs, err = translate(args); err != nil {
				return protocol.ErrorResultV1(
					fmt.Sprintf("Invalid arguments: %v", err),
				), nil
			}
		}

		result, err := bridge.RunCommand(ctx, cmdName, cliArgs, defaultMaxBytes)
		if err != nil {
			return protocol.ErrorResultV1(err.Error()), nil
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

		return &protocol.ToolCallResultV1{
			Content: []protocol.ContentBlockV1{
				protocol.TextContentV1(output),
			},
		}, nil
	}
}
