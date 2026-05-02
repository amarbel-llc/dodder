package mcp_dodder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/golf/command"
	"code.linenisgreat.com/dodder/go/internal/hotel/type_blobs"
	"code.linenisgreat.com/dodder/go/internal/sierra/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/tango/repo_actions"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/stack_frame"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/protocol"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/server"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/transport"
)

const defaultMaxBytes = 100_000

const mcpInstructionsCommon = `Dodder is a distributed zettelkasten and content-addressable blob store.

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
  → read dodder://objects/<id> (returns traversal links to type, tags, blob formats)

View an object's blob content:
  → read dodder://objects/<id>/blob/formats (discover available formatters)
  → read dodder://objects/<id>/blob/formats/<format-id> (render blob with formatter)

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
- dodder://objects/<id>/blob/formats → available blob formatters for this object's type
- dodder://objects/<id>/blob/formats/<format-id> → object blob rendered with formatter
- dodder://objects/<id>/markl → object markl integrity fields

Markl resources contain repo signatures, public keys, and object digests.
Most queries do not need this data — use only when verifying integrity or
provenance.
`

const mcpInstructionsWorkspace = `
## Workspace Tools

When dodder runs inside a workspace (directory with .dodder-workspace config),
these tools operate on checked-out objects:

- dodder_status — list checked-out objects with state (Recognized = modified,
  Untracked = new, Conflicted = merge conflict)
- dodder_diff — show internal vs external differences
- dodder_read_checked_out — read working copy file content
- dodder_checkin — commit working copy changes to the store

### Workspace Workflow

Inspect what changed:
  → dodder_status() → see all checked-out objects and their state
  → dodder_diff() → see what changed in modified objects

Read working copy content:
  → dodder_read_checked_out(object_id) → get current file content

Commit changes:
  → dodder_checkin(query) → commit matching objects to the store
`

var readOnlyAnnotations = &protocol.ToolAnnotations{
	ReadOnlyHint:   new(true),
	IdempotentHint: new(true),
}

var writeAnnotations = &protocol.ToolAnnotations{
	ReadOnlyHint:    new(false),
	DestructiveHint: new(false),
}

func RunServer(utility command.Utility, repo *local_working_copy.Repo) error {
	bridge := MakeBridge(utility)
	tools := server.NewToolRegistryV1()
	resources := server.NewResourceRegistry()
	index := makeTypeIndex(bridge)
	tagIdx := makeTagIndex(bridge)

	typeBlobCoder := type_blobs.MakeTypeStore(repo.GetEnvRepo())

	provider := &typeResourceProvider{
		registry:      resources,
		index:         index,
		tagIndex:      tagIdx,
		bridge:        bridge,
		store:         repo.GetStore(),
		typeBlobCoder: typeBlobCoder,
	}

	hasWorkspace := !repo.GetEnvWorkspace().IsTemporary()

	instructions := mcpInstructionsCommon
	if hasWorkspace {
		instructions += mcpInstructionsWorkspace
	}

	registerTools(tools, bridge, repo, index, tagIdx, hasWorkspace)
	registerResources(resources, index, tagIdx, bridge)

	prompts := server.NewPromptRegistry()
	registerPrompts(prompts)

	t := transport.NewStdio(os.Stdin, os.Stdout)
	srv, err := server.New(t, server.Options{
		ServerName:    "dodder",
		ServerVersion: "0.1.0",
		Tools:         tools,
		Resources:     provider,
		Prompts:       prompts,
		Instructions:  instructions,
	})
	if err != nil {
		return err
	}

	return srv.Run(context.Background())
}

func registerTools(tools *server.ToolRegistryV1, bridge Bridge, repo *local_working_copy.Repo, index *typeIndex, tagIdx *tagIndex, hasWorkspace bool) {
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
						"enum": ["log", "text", "json", "json-with-blob_string", "organize", "box"]
					},
					"limit": {
						"type": "integer",
						"description": "Maximum number of results to return. Defaults to 0 (unlimited). Use this to avoid large result sets when you only need a few objects."
					}
				},
				"required": ["query"],
				"additionalProperties": false
			}`),
			Annotations: readOnlyAnnotations,
		},
		func(
			ctx context.Context,
			args json.RawMessage,
		) (*protocol.ToolCallResultV1, error) {
			var p struct {
				Query  []string `json:"query"`
				Format string   `json:"format"`
				Limit  int      `json:"limit"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return protocol.ErrorResultV1(
					fmt.Sprintf("Invalid arguments: %v", err),
				), nil
			}

			format := p.Format
			if format == "" {
				format = "json"
			}

			cliArgs := []string{"-format", format}
			cliArgs = append(cliArgs, p.Query...)

			result, err := bridge.RunCommand(ctx, "show", cliArgs, defaultMaxBytes)
			if err != nil {
				errMsg := formatErrorDetail(err)
				if result.Stderr != "" {
					errMsg += "\n\nstderr:\n" + result.Stderr
				}
				return protocol.ErrorResultV1(errMsg), nil
			}

			output := result.Stdout

			if p.Limit > 0 {
				lines := strings.SplitN(output, "\n", p.Limit+1)
				if len(lines) > p.Limit {
					output = strings.Join(lines[:p.Limit], "\n")
					output += fmt.Sprintf(
						"\n\n[limited: showed %d of %d+ results]",
						p.Limit, p.Limit,
					)
				}
			}

			if result.Truncated {
				output += fmt.Sprintf(
					"\n\n[truncated: showed %d of %d bytes]",
					len(result.Stdout),
					result.BytesSeen,
				)
			}

			return &protocol.ToolCallResultV1{
				Content: []protocol.ContentBlockV1{
					protocol.TextContentV1(output),
				},
			}, nil
		},
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
			cliArgs := []string{p.ObjectId}
			if p.FormatId != "" {
				cliArgs = append(cliArgs, p.FormatId)
			}
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

	tools.Register(
		protocol.ToolV1{
			Name:        "dodder_new",
			Description: "Create a new zettel. Returns the created object in box format. Optionally set a description, type, and tags.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"description": {
						"type": "string",
						"description": "Description for the new zettel"
					},
					"tags": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Tags to apply (e.g. ['todo', 'priority-0_must'])"
					},
					"type": {
						"type": "string",
						"description": "Object type (e.g. '!md', '!task')"
					}
				},
				"additionalProperties": false
			}`),
			Annotations: writeAnnotations,
		},
		makeBridgeHandler(bridge, "new", func(args json.RawMessage) ([]string, error) {
			var p struct {
				Description string   `json:"description"`
				Tags        []string `json:"tags"`
				Type        string   `json:"type"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			var cliArgs []string
			if p.Description != "" {
				cliArgs = append(cliArgs, "-description", p.Description)
			}
			if len(p.Tags) > 0 {
				cliArgs = append(cliArgs, "-tags", strings.Join(p.Tags, ","))
			}
			if p.Type != "" {
				cliArgs = append(cliArgs, "-type", p.Type)
			}
			return cliArgs, nil
		}),
	)

	tools.Register(
		protocol.ToolV1{
			Name:        "dodder_edit",
			Description: "Edit an existing dodder object. Updates metadata (description, tags, type) and/or blob content. Fields not provided are left unchanged. When tags is provided, it replaces all existing tags.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"object_id": {
						"type": "string",
						"description": "Object identifier (e.g. zettel ID like 'ceroplastes/midtown', tag like 'todo', or type like '!md')"
					},
					"description": {
						"type": "string",
						"description": "New description (replaces existing)"
					},
					"tags": {
						"type": "array",
						"items": {"type": "string"},
						"description": "New tag set (replaces all existing tags)"
					},
					"type": {
						"type": "string",
						"description": "New object type (e.g. '!md', '!task')"
					},
					"blob": {
						"type": "string",
						"description": "New blob content (replaces existing blob)"
					}
				},
				"required": ["object_id"],
				"additionalProperties": false
			}`),
			Annotations: writeAnnotations,
		},
		makeEditHandler(repo, bridge),
	)

	if !hasWorkspace {
		// Workspace-scoped tools (status, checkin, diff, read_checked_out)
		// operate on checked-out objects in a dodder workspace. Without a
		// workspace they either assert inside the CLI command (status) or
		// return empty/unanchored results (checkin, diff, read_checked_out).
		// Advertising them anyway would mislead clients, so we simply skip
		// registration — see
		// github.com/amarbel-llc/dodder/issues/116.
		return
	}

	tools.Register(
		protocol.ToolV1{
			Name:        "dodder_status",
			Description: "List checked-out objects in the workspace with their state (CheckedOut, Recognized, Untracked, Conflicted). Requires an active workspace. Returns box format with state headers.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Optional query terms to filter objects. Defaults to all checked-out objects."
					}
				},
				"additionalProperties": false
			}`),
			Annotations: readOnlyAnnotations,
		},
		makeWorkspaceBridgeHandler(bridge, "status", func(args json.RawMessage) ([]string, error) {
			var p struct {
				Query []string `json:"query"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			return p.Query, nil
		}),
	)

	tools.Register(
		protocol.ToolV1{
			Name:        "dodder_checkin",
			Description: "Commit working copy changes to the store. Checks in objects matching the query from the workspace. Requires an active workspace.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Query terms selecting which objects to check in (e.g. [':z'] for all zettels, ['ceroplastes/midtown'] for a specific object)"
					}
				},
				"required": ["query"],
				"additionalProperties": false
			}`),
			Annotations: writeAnnotations,
		},
		makeWorkspaceBridgeHandler(bridge, "checkin", func(args json.RawMessage) ([]string, error) {
			var p struct {
				Query []string `json:"query"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			return p.Query, nil
		}),
	)

	tools.Register(
		protocol.ToolV1{
			Name:        "dodder_diff",
			Description: "Show differences between internal (store) and external (working copy) versions of checked-out objects. Requires an active workspace.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Optional query terms to filter objects. Defaults to all checked-out objects."
					}
				},
				"additionalProperties": false
			}`),
			Annotations: readOnlyAnnotations,
		},
		makeWorkspaceBridgeHandlerEmptyMessage(bridge, "diff", "no differences", func(args json.RawMessage) ([]string, error) {
			var p struct {
				Query []string `json:"query"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			return p.Query, nil
		}),
	)

	tools.Register(
		protocol.ToolV1{
			Name:        "dodder_read_checked_out",
			Description: "Read the working copy file content of a checked-out object. Returns the external (filesystem) version, not the store version. Requires an active workspace.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"object_id": {
						"type": "string",
						"description": "Object identifier (e.g. 'ceroplastes/midtown')"
					}
				},
				"required": ["object_id"],
				"additionalProperties": false
			}`),
			Annotations: readOnlyAnnotations,
		},
		makeWorkspaceBridgeHandler(bridge, "show", func(args json.RawMessage) ([]string, error) {
			var p struct {
				ObjectId string `json:"object_id"`
			}
			if err := json.Unmarshal(args, &p); err != nil {
				return nil, err
			}
			// . sigil as separate arg selects external (working copy) version
			return []string{"-format", "text", ".", p.ObjectId}, nil
		}),
	)
}

func makeEditHandler(
	repo *local_working_copy.Repo,
	bridge Bridge,
) server.ToolHandlerV1 {
	return func(
		ctx context.Context,
		args json.RawMessage,
	) (*protocol.ToolCallResultV1, error) {
		var p struct {
			ObjectId    string   `json:"object_id"`
			Description *string  `json:"description"`
			Tags        []string `json:"tags"`
			Type        *string  `json:"type"`
			Blob        *string  `json:"blob"`
		}

		if err := json.Unmarshal(args, &p); err != nil {
			return protocol.ErrorResultV1(
				fmt.Sprintf("Invalid arguments: %v", err),
			), nil
		}

		objectId, objectIdRepool, err := ids.MakeObjectId(p.ObjectId)
		if err != nil {
			return protocol.ErrorResultV1(
				fmt.Sprintf("Invalid object ID %q: %v", p.ObjectId, err),
			), nil
		}

		defer objectIdRepool()

		op := repo_actions.MakeUpdateObject(repo)

		changes := repo_actions.ObjectChanges{
			Description: p.Description,
			Tags:        p.Tags,
			Type:        p.Type,
			Blob:        p.Blob,
		}

		if _, err := op.Run(objectId, changes); err != nil {
			return protocol.ErrorResultV1(formatErrorDetail(err)), nil
		}

		// Show the updated object via bridge
		result, err := bridge.RunCommand(ctx, "show", []string{p.ObjectId}, defaultMaxBytes)
		if err != nil {
			return protocol.ErrorResultV1(formatErrorDetail(err)), nil
		}

		return &protocol.ToolCallResultV1{
			Content: []protocol.ContentBlockV1{
				protocol.TextContentV1(result.Stdout),
			},
		}, nil
	}
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
			n := min(len(children), maxErrors)

			var msg strings.Builder
			msg.WriteString(fmt.Sprintf("%s (type: %T)", err.Error(), err))
			for i := 0; i < n; i++ {
				msg.WriteString("\n  - " + describeError(children[i]))
			}
			if len(children) > maxErrors {
				msg.WriteString(fmt.Sprintf("\n  ... and %d more", len(children)-maxErrors))
			}

			return msg.String()
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
		var msg strings.Builder
		msg.WriteString(err.Error())
		frames := tree.GetErrorsAndFrames()
		for i, ef := range frames {
			if i >= 5 {
				msg.WriteString(fmt.Sprintf("\n    ... and %d more frames", len(frames)-5))
				break
			}
			msg.WriteString(fmt.Sprintf("\n    %s: %s", ef.Frame, ef.Err))
		}
		return msg.String()
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

func makeWorkspaceBridgeHandlerEmptyMessage(
	bridge Bridge,
	cmdName string,
	emptyMessage string,
	translate paramTranslator,
) server.ToolHandlerV1 {
	inner := makeWorkspaceBridgeHandler(bridge, cmdName, translate)
	return func(
		ctx context.Context,
		args json.RawMessage,
	) (*protocol.ToolCallResultV1, error) {
		result, err := inner(ctx, args)
		if err != nil {
			return result, err
		}

		if len(result.Content) == 1 && result.Content[0].Text == "" {
			result.Content[0].Text = emptyMessage
		}

		return result, nil
	}
}

func makeWorkspaceBridgeHandler(
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
