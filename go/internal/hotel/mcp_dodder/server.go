package mcp_dodder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/lib/_/stack_frame"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/protocol"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/server"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/transport"
)

const defaultMaxBytes = 100_000

const mcpInstructions = `Dodder is a distributed zettelkasten and content-addressable blob store.

## Object Listings — Box Format

Object listings (e.g. dodder://types/<id>/objects) use the compact box format.
Each line represents one object:

  [<object-id> @<blob-digest> !<type> <tag1> <tag2> ...] <description>

Field order inside brackets:
1. Object ID (e.g. thallium/golem, !md, konfig)
2. Blob digest prefixed with @ (e.g. @blake2b256-9ft3...)
3. Type prefixed with ! (e.g. !md, !toml-type-v1)
4. Tags as bare identifiers, sorted alphabetically. Tags prefixed with %
   are auto-generated (by the object's type or other entities).
Description appears as a trailer after the closing bracket.

Values containing spaces are Go-quoted ("like this").

Examples:

  [thallium/golem !task area-home urgency-2_week] purchase izipizi glasses
  [ceroplastes/midtown @blake2b256-9ft3... !md project-2024-q3] meeting notes
  [!md @blake2b256-76m5... !toml-type-v1]

## Resource Drill-Down

- dodder://types_index → word list for search
- dodder://types → all type summaries
- dodder://types/<id> → type metadata + links to sub-resources
- dodder://types/<id>/objects → all objects of this type (box format)
- dodder://types/<id>/blob → type blob content (TOML config)
- dodder://objects/<id>/blob/<format> → object blob rendered with formatter
- dodder://types/<id>/markl → type markl (merkle-tree) integrity fields
- dodder://objects/<id>/markl → object markl integrity fields

Markl resources contain repo signatures, public keys, and object digests.
Most queries do not need this data — use only when verifying integrity or
provenance.
`

var readOnlyAnnotations = &protocol.ToolAnnotations{
	ReadOnlyHint:   protocol.BoolPtr(true),
	IdempotentHint: protocol.BoolPtr(true),
}

func RunServer(utility command.Utility) error {
	bridge := MakeBridge(utility)
	tools := server.NewToolRegistryV1()
	resources := server.NewResourceRegistry()
	index := makeTypeIndex(bridge)

	provider := &typeResourceProvider{
		registry: resources,
		index:    index,
		bridge:   bridge,
	}

	registerTools(tools, bridge, index)
	registerResources(resources, index, bridge)

	t := transport.NewStdio(os.Stdin, os.Stdout)
	srv, err := server.New(t, server.Options{
		ServerName:    "dodder",
		ServerVersion: "0.1.0",
		Tools:         tools,
		Resources:     provider,
		Instructions:  mcpInstructions,
	})
	if err != nil {
		return err
	}

	return srv.Run(context.Background())
}

func registerTools(tools *server.ToolRegistryV1, bridge Bridge, index *typeIndex) {
	tools.Register(
		protocol.ToolV1{
			Name:        "dodder_show",
			Description: "View a specific dodder object by ID. Returns metadata and content for zettels, tags, or types.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"object_id": {
						"type": "string",
						"description": "Object identifier (e.g. zettel ID like 'ceroplastes/midtown', tag like '%tag', or type like '!type')"
					},
					"format": {
						"type": "string",
						"description": "Output format (log, text, json, organize). Defaults to log.",
						"enum": ["log", "text", "json", "organize"]
					}
				},
				"required": ["object_id"],
				"additionalProperties": false
			}`),
			Annotations: readOnlyAnnotations,
		},
		makeBridgeHandler(bridge, "show", func(args json.RawMessage) ([]string, error) {
			var p struct {
				ObjectId string `json:"object_id"`
				Format   string `json:"format"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			var cliArgs []string
			if p.Format != "" {
				cliArgs = append(cliArgs, "-format", p.Format)
			}
			cliArgs = append(cliArgs, p.ObjectId)
			return cliArgs, nil
		}),
	)

	tools.Register(
		protocol.ToolV1{
			Name:        "dodder_query",
			Description: "Search for dodder objects matching a query expression. Query terms are combined with AND. Examples: ':z' (all zettels), ':t' (all tags), '%todo' (tagged with todo), '!article' (type article).",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Query terms (e.g. [':z', '%todo'] for zettels tagged todo)"
					},
					"format": {
						"type": "string",
						"description": "Output format. Defaults to json (without blob content). Use json-with-blob_string to include blob content.",
						"enum": ["log", "text", "json", "json-with-blob_string", "organize"]
					}
				},
				"required": ["query"],
				"additionalProperties": false
			}`),
			Annotations: readOnlyAnnotations,
		},
		makeBridgeHandler(bridge, "show", func(args json.RawMessage) ([]string, error) {
			var p struct {
				Query  []string `json:"query"`
				Format string   `json:"format"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			format := p.Format
			if format == "" {
				format = "json"
			}
			cliArgs := []string{"-format", format}
			cliArgs = append(cliArgs, p.Query...)
			return cliArgs, nil
		}),
	)

	tools.Register(
		protocol.ToolV1{
			Name:        "dodder_format_blob",
			Description: "Format and display the blob content of a dodder zettel. Renders the zettel's associated file content using the type's configured formatter.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"object_id": {
						"type": "string",
						"description": "Object identifier for the zettel whose blob to format"
					},
					"format_id": {
						"type": "string",
						"description": "Formatter ID to use (optional, uses type default if omitted)"
					}
				},
				"required": ["object_id"],
				"additionalProperties": false
			}`),
			Annotations: readOnlyAnnotations,
		},
		makeBridgeHandler(bridge, "format-blob", func(args json.RawMessage) ([]string, error) {
			var p struct {
				ObjectId string `json:"object_id"`
				FormatId string `json:"format_id"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			var cliArgs []string
			if p.FormatId != "" {
				cliArgs = append(cliArgs, p.FormatId)
			}
			cliArgs = append(cliArgs, p.ObjectId)
			return cliArgs, nil
		}),
	)

	tools.Register(
		protocol.ToolV1{
			Name:        "dodder_type_query",
			Description: "Search for dodder types by word. Words are matched against type IDs, descriptions, and tags (all expanded by hyphen segments). Returns compact summaries with resource URIs for drill-down. Use dodder://types_index resource to discover available words.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"words": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Search words (OR-union). E.g. ['task', 'md'] matches types containing either word."
					}
				},
				"required": ["words"],
				"additionalProperties": false
			}`),
			Annotations: readOnlyAnnotations,
		},
		func(
			ctx context.Context,
			args json.RawMessage,
		) (*protocol.ToolCallResultV1, error) {
			var p struct {
				Words []string `json:"words"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return protocol.ErrorResultV1(
					fmt.Sprintf("Invalid arguments: %v", err),
				), nil
			}

			if err := index.ensureBuilt(); err != nil {
				return protocol.ErrorResultV1(
					fmt.Sprintf("Failed to build type index: %v", err),
				), nil
			}

			results := index.query(p.Words)

			output, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				return protocol.ErrorResultV1(err.Error()), nil
			}

			return &protocol.ToolCallResultV1{
				Content: []protocol.ContentBlockV1{
					protocol.TextContentV1(string(output)),
				},
			}, nil
		},
	)
}

func formatErrorDetail(err error) string {
	type unwrapMany interface {
		Unwrap() []error
	}

	// Unwrap through single-error wrappers to find a multi-error
	current := err
	for current != nil {
		if group, ok := current.(unwrapMany); ok {
			children := group.Unwrap()
			const maxErrors = 3
			n := len(children)
			if n > maxErrors {
				n = maxErrors
			}

			msg := fmt.Sprintf("%s (type: %T)", err.Error(), err)
			for i := 0; i < n; i++ {
				msg += "\n  - " + describeError(children[i])
			}
			if len(children) > maxErrors {
				msg += fmt.Sprintf("\n  ... and %d more", len(children)-maxErrors)
			}

			return msg
		}

		type unwrapOne interface {
			Unwrap() error
		}

		if w, ok := current.(unwrapOne); ok {
			current = w.Unwrap()
		} else {
			break
		}
	}

	return fmt.Sprintf("%s (type: %T)", err.Error(), err)
}

func describeError(err error) string {
	var tree *stack_frame.ErrorTree
	if ok := errors.As(err, &tree); ok {
		msg := err.Error()
		frames := tree.GetErrorsAndFrames()
		for i, ef := range frames {
			if i >= 5 {
				msg += fmt.Sprintf("\n    ... and %d more frames", len(frames)-5)
				break
			}
			msg += fmt.Sprintf("\n    %s: %s", ef.Frame, ef.Err)
		}
		return msg
	}

	return fmt.Sprintf("[%T] %s", err, err.Error())
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
			errMsg := formatErrorDetail(err)
			if result.Stderr != "" {
				errMsg += "\n\nstderr:\n" + result.Stderr
			}
			return protocol.ErrorResultV1(errMsg), nil
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
