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

## Data Model

Every object in dodder has: an object-id, a date, an optional description,
an optional type, and zero or more tags. Tags are themselves objects that
can have their own tags (meta-tags). Common meta-tag patterns:

- active — marks a project/tag as currently active
- priority-0_must, priority-1_should, priority-2_want — priority levels
- area-home, area-career, area-health — life areas
- project-* — project groupings

Object genres:
- Zettels: ID has left/right parts separated by / (e.g. thallium/golem)
- Types: ID prefixed with ! (e.g. !task, !md)
- Tags: bare identifier, no ! prefix, no / separator (e.g. priority-0_must)

## Query Syntax

Query terms in dodder_query are AND-combined. Term types:
- Genre filters: :z (zettels), :e (tags), :t (types)
- Tag filter: bare tag name (e.g. todo, priority-0_must)
- Type filter: !type (e.g. !task, !md)

Examples: [":z", "todo"] = zettels tagged todo. ["!task", "urgency-2_week"] =
tasks with urgency-2_week tag. [":e"] = all tag objects.

## Tool Selection Guide

1. type_query / tag_query — START HERE for discovery. Returns summaries with
   tags and resource URIs. Use tag_query to find tags by word (e.g. ["project"]
   finds all project-* tags), then inspect the tags field to filter
   (e.g. check for "active" in tags).

2. Resources (dodder://...) — DRILL DOWN for detail. Follow resource URIs from
   query results. Use /objects for listings, /objects/facets for analytics.

3. dodder_query — RAW QUERIES when you need AND-combined filters or specific
   format output. Returns full object data.

4. dodder_show — VIEW A SINGLE OBJECT by exact ID.

## Common Workflows

Find active projects:
  → tag_query(["project"]) → filter results where tags contains "active"

Find tasks by priority:
  → dodder_query(["!task", "priority-0_must"]) with format "box"

Summarize a type's tag distribution:
  → read dodder://types/<id>/objects/facets

Browse objects of a type:
  → read dodder://types/<id>/objects (box format listing)

Inspect a specific object and navigate to related objects:
  → read dodder://objects/<id> (returns traversal links to type, tags, blob)

## Object Listings — Box Format

Object listings (e.g. dodder://types/<id>/objects) use the compact box format.
Each line represents one object:

  [<object-id> @<blob-digest> !<type> <tag1> <tag2> ...] <description>

Field order inside brackets:
1. Object ID (e.g. thallium/golem, !md, konfig)
2. Blob digest prefixed with @ (e.g. @blake2b256-9ft3...)
3. Type prefixed with ! (e.g. !md, !toml-type-v1)
4. Tags as bare identifiers, sorted alphabetically. Tags prefixed with %
   are not persisted to the repo (computed/derived at display time by the
   object's type or other entities).
Description appears as a trailer after the closing bracket.

Values containing spaces are Go-quoted ("like this").

Examples:

  [thallium/golem !task area-home urgency-2_week] purchase izipizi glasses
  [ceroplastes/midtown @blake2b256-9ft3... !md project-2024-q3] meeting notes
  [!md @blake2b256-76m5... !toml-type-v1]

## Resource Drill-Down

### Types
- dodder://types_index → word list for search
- dodder://types → all type summaries
- dodder://types/<id> → type metadata + links to sub-resources
- dodder://types/<id>/objects → all objects of this type (box format)
- dodder://types/<id>/objects/facets → tag breakdown grouped by prefix
- dodder://types/<id>/blob → type blob content (TOML config)
- dodder://types/<id>/markl → type markl (merkle-tree) integrity fields

### Tags
- dodder://tags_index → word list for search
- dodder://tags → all tag summaries
- dodder://tags/<id> → tag metadata + links to sub-resources
- dodder://tags/<id>/objects → all objects with this tag (box format)
- dodder://tags/<id>/objects/facets → tag breakdown grouped by prefix
- dodder://tags/<id>/markl → tag markl (merkle-tree) integrity fields

### Objects
- dodder://objects/<id> → object metadata + traversal links to type, tags, blob, markl
- dodder://objects/<id>/blob/<format> → object blob rendered with formatter
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
	tagIdx := makeTagIndex(bridge)

	provider := &typeResourceProvider{
		registry: resources,
		index:    index,
		tagIndex: tagIdx,
		bridge:   bridge,
	}

	registerTools(tools, bridge, index, tagIdx)
	registerResources(resources, index, tagIdx, bridge)

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

func registerTools(tools *server.ToolRegistryV1, bridge Bridge, index *typeIndex, tagIdx *tagIndex) {
	tools.Register(
		protocol.ToolV1{
			Name:        "dodder_show",
			Description: "View a specific dodder object by ID. Returns metadata and content for zettels, tags, or types.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"object_id": {
						"type": "string",
						"description": "Object identifier (e.g. zettel ID like 'ceroplastes/midtown', tag like 'todo', or type like '!type')"
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
			Description: "Search for dodder objects matching a query expression. Query terms are AND-combined. Term types: genre filters (:z zettels, :e tags, :t types), tag filters (bare name like 'todo'), type filters (!task). Examples: [':z', 'todo'] = zettels tagged todo, ['!task', 'priority-0_must'] = must-do tasks. Prefer type_query/tag_query for discovery; use this for AND-filtered object listings.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Query terms (e.g. [':z', 'todo'] for zettels tagged todo)"
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
			Description: "Search for dodder types by word (OR-union). Returns type summaries including tags and resource URIs for drill-down. Words are expanded by hyphen segments: 'task' matches !task, !taskpaper, etc. Use dodder://types/<id>/objects to list objects of a matched type, or /objects/facets for tag analytics.",
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

	tools.Register(
		protocol.ToolV1{
			Name:        "dodder_tag_query",
			Description: "Search for dodder tags by word (OR-union). Returns tag summaries including each tag's own tags (meta-tags like 'active', 'priority-0_must'). Use this to discover and filter tags — e.g. tag_query(['project']) returns all project tags, then check each result's tags field for 'active' to find active projects. Words are expanded by hyphen segments: 'project' matches project-2021-zit, project-24q2-personal_sites, etc.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"words": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Search words (OR-union). E.g. ['priority', 'urgency'] matches tags containing either word."
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

			if err := tagIdx.ensureBuilt(); err != nil {
				return protocol.ErrorResultV1(
					fmt.Sprintf("Failed to build tag index: %v", err),
				), nil
			}

			results := tagIdx.query(p.Words)

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
