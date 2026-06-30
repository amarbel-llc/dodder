package mcp_dodder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/alfa/mcp_tool_perms"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/repo_id"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/juliett/queries"
	"code.linenisgreat.com/dodder/go/internal/kilo/orgie"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	"code.linenisgreat.com/dodder/go/internal/sierra/repo_actions"
	"github.com/amarbel-llc/madder/go/pkgs/scoped_id"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/stack_frame"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/protocol"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/server"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/transport"
)

const defaultMaxBytes = 100_000

const mcpInstructionsCommon = `Dodder is a distributed zettelkasten and content-addressable blob store.

## Data Model

Every object in dodder has: an object-id, a date, an optional description,
a type, and zero or more tags. Tags are themselves objects that
can have their own tags (meta-tags). Common meta-tag patterns:

- active — marks a project/tag as currently active
- priority-0_must, priority-1_should, priority-2_want — priority levels
- area-home, area-career, area-health — life areas
- project-* — project groupings

Object genres:
- Zettels: ID has left/right parts separated by / (e.g. thallium/golem)
- Types: ID prefixed with ! (e.g. !task, !md)
- Tags: bare identifier, no ! prefix, no / separator (e.g. priority-0_must)

Types and the default type: a zettel's type is supplied by the repo's
DEFAULT TYPE when you omit one on the new tool. Workspace repos and clones
have NO default type, so creating a zettel there REQUIRES an explicit type —
omitting it errors with "no type given and repo has no default type; pass
-type" (set the new tool's "type" argument). Existing or imported data may
also be typeless (it reads as a bare !); typeless objects break push/import,
so always give a new zettel a type.

## Query Syntax

Query terms in the query tool are AND-combined. Term types:
- Genre filters: :z (zettels), :e (tags), :t (types)
- Tag filter: bare tag name (e.g. todo, priority-0_must)
- Type filter: !type (e.g. !task, !md)

Examples: [":z", "todo"] = zettels tagged todo. ["!task", "urgency-2_week"] =
tasks with urgency-2_week tag. [":e"] = all tag objects.

Sigils modify selection scope. A sigil attaches to the query TERM (not as a
prefix), immediately before the genre letter:
- : latest version only (default)
- + include historical versions (e.g. !task+:t — NOT +!task:t)
- . include external / checked-out objects
- ? include dormant / hidden objects
Sigils combine: ":." = latest + external. Putting a sigil at the front of a
term (e.g. +!task:t) is a syntax error.

Tag matching is TRANSITIVE through meta-tags. A tag filter matches an object
tagged with that tag directly OR tagged with any tag whose own meta-tags
include it. Example: if the tag "project-recurse" is meta-tagged "area-career",
then a zettel tagged "project-recurse" matches ["area-career", ":z"] even
though "area-career" is not in that zettel's own tag list. So an object can
correctly appear in a tag query without carrying the queried tag directly —
check the meta-tags of its tags (the tag's own tags field) before concluding a
result is wrong. The matching surface is the expanded tag closure, not the
direct tag set.

## Tool Selection Guide

1. query-type / query-tag — START HERE for discovery. Returns summaries with
   tags and resource URIs. Use query-tag to find tags by word (e.g. ["project"]
   finds all project-* tags), then inspect the tags field to filter
   (e.g. check for "active" in tags).

2. Resources (dodder://...) — DRILL DOWN for detail. Follow resource URIs from
   query results. Use /objects for listings, /objects/facets for analytics.

3. query — RAW QUERIES when you need AND-combined filters or specific
   format output. Returns full object data.

4. show — VIEW A SINGLE OBJECT by exact ID.

## Common Workflows

Find active projects:
  → query-tag(["project"]) → filter results where tags contains "active"

Find tasks by priority:
  → query(["!task", "priority-0_must"]) with format "box"

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

## Remote Transfer

These tools move objects between repos (no workspace required):
- import(paths) — import objects from inventory list files into the local store
- push(remote?, query?, direct?) — send objects to a remote (stored remote id,
  a direct local path, or the workspace's pinned parent when both are omitted)
- pull(remote?, query?, direct?) — fetch objects from a remote into the local store

## Resource Drill-Down

The un-segmented dodder://<kind>/... URIs below address the server's
own repo (CWD-auto sugar). To address a specific repo without
restarting the server, prefix any path with repos/<repo>: e.g.
dodder:///repos/<repo>/types/<id>, dodder:///repos/<repo>/objects/<id>.
Read dodder:///repos to list the repos in scope, and
dodder:///repos/<repo> for a per-repo overview. <repo> is the repo's
CLI spelling (e.g. work, .work). Traversal links in repo-scoped reads
stay within the addressed repo.

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

- status — list checked-out objects with state (Recognized = modified,
  Untracked = new, Conflicted = merge conflict)
- diff — show internal vs external differences
- read-checked_out — read working copy file content
- checkin — commit working copy changes to the store

### Workspace Workflow

Inspect what changed:
  → status() → see all checked-out objects and their state
  → diff() → see what changed in modified objects

Read working copy content:
  → read-checked_out(object_id) → get current file content

Commit changes:
  → checkin(query) → commit matching objects to the store
`

var readOnlyAnnotations = &protocol.ToolAnnotations{
	ReadOnlyHint:   new(true),
	IdempotentHint: new(true),
}

var writeAnnotations = &protocol.ToolAnnotations{
	ReadOnlyHint:    new(false),
	DestructiveHint: new(false),
}

// destructiveAnnotations marks tools that discard state (reset-lock).
// The clown plugin's PreToolUse hook additionally forces a user prompt
// on every call to such tools (#249).
var destructiveAnnotations = &protocol.ToolAnnotations{
	ReadOnlyHint:    new(false),
	DestructiveHint: new(true),
}

// annotationsFor maps a tool's shared permission classification
// (mcp_tool_perms, the single source of truth shared with the clown
// plugin hook, #251) to the go-mcp annotations registered with it. An
// unclassified name fails safe as a write tool (no read-only/idempotent
// hint); the consistency test asserts every registered tool is
// classified, so that branch is unreachable in practice.
func annotationsFor(name string) *protocol.ToolAnnotations {
	switch mcp_tool_perms.Of(name) {
	case mcp_tool_perms.PermissionReadOnly:
		return readOnlyAnnotations
	case mcp_tool_perms.PermissionDestructive:
		return destructiveAnnotations
	default:
		return writeAnnotations
	}
}

// registerTool registers a tool with the annotations its shared
// permission classification dictates (mcp_tool_perms), so the
// annotation is never set by hand at the call site and cannot drift
// from the classification the clown plugin hook reads (#251).
func registerTool(
	reg *server.ToolRegistryV1,
	tool protocol.ToolV1,
	handler server.ToolHandlerV1,
) {
	tool.Annotations = annotationsFor(tool.Name)
	reg.Register(tool, handler)
}

func RunServer(
	utility command.Utility,
	repo *local_working_copy.Repo,
	startupRepoId scoped_id.Id,
	userReposDir string,
	dodderVersion string,
) error {
	// An explicit startup -repo_id pins the server to that repo (per-call
	// repo_id is then restricted to it); an auto/default startup lets each
	// tool call select a repo freely (FDR-0019).
	pinned := !repo_id.IsAuto(startupRepoId)
	bridge := MakeBridge(utility, startupRepoId, pinned)
	tools := server.NewToolRegistryV1()
	resources := server.NewResourceRegistry()

	// The startup repo's indexes back the query-type / query-tag tools and
	// the auto/default resource reads. Seed the provider's per-repo maps
	// with them (keyed by the startup repo's segment) so a tool build and a
	// resource read of the same repo share one lazily-built index.
	index := makeTypeIndex(bridge, startupRepoId)
	tagIdx := makeTagIndex(bridge, startupRepoId)

	// reposDir is the un-nested <data>/repos/ directory. A named repo nests
	// its metadata data dir under repos/<name>/ (madder#241), so the parent
	// of the startup repo's data dir is the repos/ collection that
	// readReposList scans. Every repo (including the default) is named, so
	// this parent is always the repos/ dir.
	reposDir := filepath.Dir(repo.GetEnvRepo().GetXDG().Data.ActualValue)

	// IsOverridden reports a cwd ancestor .dodder/ scope (spelled `.name`)
	// versus the XDG user home (spelled `name`) — drives the repo listing's
	// -repo_id spelling (FDR-0019 #276).
	startupIsCwd := repo.GetEnvRepo().GetXDG().IsOverridden()

	startupSeg := repoSeg(startupRepoId)

	provider := &typeResourceProvider{
		registry:     resources,
		bridge:       bridge,
		reposDir:     reposDir,
		startupIsCwd: startupIsCwd,
		userReposDir: userReposDir,
		typeIndexes:  map[string]*typeIndex{startupSeg: index},
		tagIndexes:   map[string]*tagIndex{startupSeg: tagIdx},
	}

	hasWorkspace := !repo.GetEnvWorkspace().IsTemporary()

	instructions := mcpInstructionsCommon
	if hasWorkspace {
		instructions += mcpInstructionsWorkspace
	}

	registerTools(tools, bridge, index, tagIdx, hasWorkspace)
	registerResources(resources, provider)

	prompts := server.NewPromptRegistry()
	registerPrompts(prompts, provider, startupRepoId, hasWorkspace, dodderVersion)

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

func registerTools(tools *server.ToolRegistryV1, bridge Bridge, index *typeIndex, tagIdx *tagIndex, hasWorkspace bool) {
	registerTool(
		tools,
		protocol.ToolV1{
			Name:        "show",
			Description: "View a specific dodder object by ID. Returns metadata and content for zettels, tags, or types.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"object_id": {
						"type": "string",
						"description": "Object identifier or query term (e.g. zettel ID like 'ceroplastes/midtown', tag like 'todo', type like '!type', or a genre query like ':z'). Sigils attach to the TERM before the genre letter: ':' latest (default), '+' history (e.g. '!task+:t', NOT '+!task:t'), '.' external/checked-out, '?' dormant/hidden."
					},
					"format": {
						"type": "string",
						"description": "Output format (log, text, json). Defaults to log.",
						"enum": ["log", "text", "json"]
					},
					"repo_id": {
						"type": "string",
						"description": "Repo to operate on: name (XDG user) or .name (current dir). Defaults to the server's repo."
					}
				},
				"required": ["object_id"],
				"additionalProperties": false
			}`),
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

	registerTool(
		tools,
		protocol.ToolV1{
			Name:        "query",
			Description: "Search for dodder objects matching a query expression. Query terms are AND-combined. Term types: genre filters (:z zettels, :e tags, :t types), tag filters (bare name like 'todo'), type filters (!task). Examples: [':z', 'todo'] = zettels tagged todo, ['!task', 'priority-0_must'] = must-do tasks. Sigils attach to the TERM before the genre letter: ':' latest (default), '+' history (e.g. '!task+:t', NOT '+!task:t'), '.' external/checked-out, '?' dormant/hidden; they combine (':.' = latest + external). Prefer query-type/query-tag for discovery; use this for AND-filtered object listings. Tag matching is transitive through meta-tags: an object matches a tag filter if it is tagged with that tag directly OR with any tag whose own meta-tags include it, so a result can lack the queried tag in its own tag list (that is correct, not a leak).",
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
						"enum": ["log", "text", "json", "json-with-blob_string", "box"]
					},
					"limit": {
						"type": "integer",
						"description": "Maximum number of results to return. Defaults to 0 (unlimited). Use this to avoid large result sets when you only need a few objects."
					},
					"repo_id": {
						"type": "string",
						"description": "Repo to operate on: name (XDG user) or .name (current dir). Defaults to the server's repo."
					}
				},
				"required": ["query"],
				"additionalProperties": false
			}`),
		},
		func(
			ctx context.Context,
			args json.RawMessage,
		) (*protocol.ToolCallResultV1, error) {
			var p struct {
				Query  []string `json:"query"`
				Format string   `json:"format"`
				Limit  int      `json:"limit"`
				RepoId string   `json:"repo_id"`
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

			result, err := bridge.RunCommandWithRepoId(
				ctx,
				"show",
				cliArgs,
				defaultMaxBytes,
				p.RepoId,
			)
			if err != nil {
				errMsg := formatToolError(err)
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

			if output == "" {
				output = emptyOutputMessage
			}

			return &protocol.ToolCallResultV1{
				Content: []protocol.ContentBlockV1{
					protocol.TextContentV1(output),
				},
			}, nil
		},
	)

	registerTool(
		tools,
		protocol.ToolV1{
			Name:        "format-blob",
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
					},
					"repo_id": {
						"type": "string",
						"description": "Repo to operate on: name (XDG user) or .name (current dir). Defaults to the server's repo."
					}
				},
				"required": ["object_id"],
				"additionalProperties": false
			}`),
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

	registerTool(
		tools,
		protocol.ToolV1{
			Name:        "query-type",
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

	registerTool(
		tools,
		protocol.ToolV1{
			Name:        "query-tag",
			Description: "Search for dodder tags by word (OR-union). Returns tag summaries including each tag's own tags (meta-tags like 'active', 'priority-0_must'). Use this to discover and filter tags — e.g. query-tag(['project']) returns all project tags, then check each result's tags field for 'active' to find active projects. Words are expanded by hyphen segments: 'project' matches project-2021-zit, project-24q2-personal_sites, etc.",
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

			// Non-silent empty result (#306): query-tag searches only
			// materialized tag OBJECTS (genre :e). Tags applied to objects as
			// plain strings are never auto-materialized, so an empty result in
			// a tag-heavy repo otherwise reads as "no match" when the real
			// reason is "no tag objects exist." Explain the contract instead of
			// returning a bare [].
			if len(results) == 0 {
				return &protocol.ToolCallResultV1{
					Content: []protocol.ContentBlockV1{
						protocol.TextContentV1(
							"No matching tag objects.\n\n" +
								"query-tag searches only materialized tag objects " +
								"(genre :e, authored via `new -object-id <tag>` or " +
								"`organize`). Tags applied to objects as plain " +
								"strings are not materialized as tag objects, so " +
								"they do not appear here and carry no meta-tags " +
								"until materialized. To filter objects by a tag " +
								"string regardless of materialization, use the " +
								"query tool with the bare tag name (e.g. query " +
								"[\"<tag>\"]). See the tag materialization behavior " +
								"in docs/features/0022-tag-materialization.md.",
						),
					},
				}, nil
			}

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

	registerTool(
		tools,
		protocol.ToolV1{
			Name:        "new",
			Description: "Create a new object. Returns the created object in box format. With no object_id, creates a zettel with an auto-assigned id. Set object_id to author a tag or a type (e.g. '!task'). Optionally set a description, type, tags, and an inline blob body. NOTE: a zettel needs a type — the repo's default type fills in when you omit one, but workspace repos and clones have NO default type, so creating a zettel there requires an explicit 'type' (omitting it errors 'no type given and repo has no default type; pass -type').",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"object_id": {
						"type": "string",
						"description": "Object id to assign. Empty (default) auto-assigns a zettel id. A zettel id like 'a/b', a tag like 'foo', or a type like '!task'. A non-zettel id sets the meta-type automatically from the id's genre, so do NOT also pass 'type' in that case."
					},
					"description": {
						"type": "string",
						"description": "Description for the new object"
					},
					"tags": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Tags to apply (e.g. ['todo', 'priority-0_must'])"
					},
					"type": {
						"type": "string",
						"description": "Object type (e.g. '!md'). For a zettel only; omit when object_id names a tag or type. Required for a zettel in a repo with no default type (workspace repos and clones)."
					},
					"blob": {
						"type": "string",
						"description": "Inline blob body for the new object (e.g. the TOML body of a '!task' type). Empty (default) writes no blob."
					},
					"repo_id": {
						"type": "string",
						"description": "Repo to create in: name (XDG user) or .name (current dir). Defaults to the server's repo."
					}
				},
				"additionalProperties": false
			}`),
		},
		makeBridgeHandler(bridge, "new", newToolCLIArgs),
	)

	registerTool(
		tools,
		protocol.ToolV1{
			Name:        "edit",
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
					},
					"repo_id": {
						"type": "string",
						"description": "Repo to operate on: name (XDG user) or .name (current dir). Defaults to the server's repo."
					}
				},
				"required": ["object_id"],
				"additionalProperties": false
			}`),
		},
		makeEditHandler(bridge),
	)

	registerTool(
		tools,
		protocol.ToolV1{
			Name:        "reset-lock",
			Description: "Forcibly reset the repo's environment lock (the filesystem mutex guarding mutations, NOT content locks). A failed mutating operation intentionally leaves this lock held for recovery; use this tool to clear it after verifying no other dodder process is using the repo. The response states the resulting lock state unambiguously. Requires user approval on every call.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"repo_id": {
						"type": "string",
						"description": "Repo whose lock to reset: name (XDG user) or .name (current dir). Defaults to the server's repo."
					}
				},
				"additionalProperties": false
			}`),
		},
		makeResetLockHandler(bridge),
	)

	registerTool(
		tools,
		protocol.ToolV1{
			Name:        "organize_plan",
			Description: "Render objects matching a query as an organize buffer for batch editing — the read half of the organize plan/commit flow. The buffer groups objects under tag headings; you edit it (add a heading to apply a tag across the objects beneath it, move an object line under a different heading to change its tags, edit an object's description, delete a line to remove a tag) and pass the WHOLE edited buffer to organize_commit with the SAME query. This is the ergonomic way to apply tag/description changes across many objects in one round-trip instead of N edit calls. Read-only: produces the buffer, changes nothing.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Query terms selecting the objects to organize (e.g. ['!task'], [':z', 'project-alpha']). Same syntax as the query tool; default genre is zettels."
					},
					"repo_id": {
						"type": "string",
						"description": "Repo to operate on: name (XDG user) or .name (current dir). Defaults to the server's repo."
					}
				},
				"required": ["query"],
				"additionalProperties": false
			}`),
		},
		makeOrganizePlanHandler(bridge),
	)

	registerTool(
		tools,
		protocol.ToolV1{
			Name:        "organize_commit",
			Description: "Apply an edited organize buffer — the write half of the organize plan/commit flow. Pass the SAME query you gave organize_plan plus the edited buffer; the changes (tag adds/removes from moving objects between headings, description edits) are applied to all affected objects in one batch. The query re-derives the baseline the edited buffer is diffed against, so it must match the organize_plan call. Returns a summary of what changed.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {
						"type": "array",
						"items": {"type": "string"},
						"description": "The SAME query terms passed to organize_plan. Re-runs to establish the before-edit baseline; a different query will diff against the wrong baseline."
					},
					"organize": {
						"type": "string",
						"description": "The full edited organize buffer (the organize_plan output with your edits)."
					},
					"repo_id": {
						"type": "string",
						"description": "Repo to operate on: name (XDG user) or .name (current dir). Defaults to the server's repo."
					}
				},
				"required": ["query", "organize"],
				"additionalProperties": false
			}`),
		},
		makeOrganizeCommitHandler(bridge),
	)

	registerTool(
		tools,
		protocol.ToolV1{
			Name:        "import",
			Description: "Import objects from inventory list files into the local store. Reads one or more inventory list files (e.g. produced by the export command) and commits the objects they describe. Blobs are pulled from the named madder blob store when blob_store_id is set. Does not require a workspace.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"paths": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Paths to inventory list files to import (at least one)."
					},
					"blob_store_id": {
						"type": "string",
						"description": "Name of an existing madder blob store to read blobs from during import."
					},
					"omit_tags": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Regex patterns for tags to strip from imported objects (repeatable)."
					},
					"dry_run": {
						"type": "boolean",
						"description": "Preview the import plan without committing. Returns the plan instead of importing; combine with plan_format."
					},
					"plan_format": {
						"type": "string",
						"description": "Format for the dry_run plan: 'summary' (default, box listing) or 'objects' (per-object classifications).",
						"enum": ["summary", "objects"]
					},
					"blobless_type_remapping": {
						"type": "object",
						"additionalProperties": {"type": "string"},
						"description": "Resolve blobless types by remapping them to local types, e.g. {\"!oldtype\": \"!md\"}. Each entry replaces the type on objects that reference a type with no blob."
					},
					"repo_id": {
						"type": "string",
						"description": "Repo to operate on: name (XDG user) or .name (current dir). Defaults to the server's repo."
					}
				},
				"required": ["paths"],
				"additionalProperties": false
			}`),
		},
		makeBridgeHandler(bridge, "import", importToolCLIArgs),
	)

	registerTool(
		tools,
		protocol.ToolV1{
			Name:        "push",
			Description: "Push objects to a remote repository. Sends matching objects (and their blobs) from the local repo to a remote. Identify the remote with 'remote' (a stored remote repository object id), 'direct' (a local repository path), or omit both to use the workspace's pinned parent. Optionally limit the transfer with query terms.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"remote": {
						"type": "string",
						"description": "Stored remote repository object id to push to (e.g. '/them'). Omit to use the workspace's pinned parent, or set 'direct' instead."
					},
					"query": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Optional query terms limiting which objects to push. Defaults to all inventory lists (full history)."
					},
					"direct": {
						"type": "string",
						"description": "Path to a local dodder repository for a direct push without a stored remote."
					},
					"repo_id": {
						"type": "string",
						"description": "Repo to operate on: name (XDG user) or .name (current dir). Defaults to the server's repo."
					}
				},
				"additionalProperties": false
			}`),
		},
		makeBridgeHandler(bridge, "push", pushPullToolCLIArgs),
	)

	registerTool(
		tools,
		protocol.ToolV1{
			Name:        "pull",
			Description: "Pull objects from a remote repository. Fetches matching objects (and their blobs) from a remote into the local repo. Identify the remote with 'remote' (a stored remote repository object id), 'direct' (a local repository path), or omit both to use the workspace's pinned parent. Optionally limit the transfer with query terms.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"remote": {
						"type": "string",
						"description": "Stored remote repository object id to pull from (e.g. '/them'). Omit to use the workspace's pinned parent, or set 'direct' instead."
					},
					"query": {
						"type": "array",
						"items": {"type": "string"},
						"description": "Optional query terms limiting which objects to pull. Defaults to all inventory lists (full history)."
					},
					"direct": {
						"type": "string",
						"description": "Path to a local dodder repository for a direct pull without a stored remote."
					},
					"repo_id": {
						"type": "string",
						"description": "Repo to operate on: name (XDG user) or .name (current dir). Defaults to the server's repo."
					}
				},
				"additionalProperties": false
			}`),
		},
		makeBridgeHandler(bridge, "pull", pushPullToolCLIArgs),
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

	registerTool(
		tools,
		protocol.ToolV1{
			Name:        "status",
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

	registerTool(
		tools,
		protocol.ToolV1{
			Name:        "checkin",
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

	registerTool(
		tools,
		protocol.ToolV1{
			Name:        "diff",
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

	registerTool(
		tools,
		protocol.ToolV1{
			Name:        "read-checked_out",
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

// newToolCLIArgs translates the `new` tool's arguments into `dodder new`
// CLI flags. -edit=false is always passed first so the command never
// launches an editor: the MCP server has no controlling TTY, and an
// editor launch blocks the tool call indefinitely (the object is
// committed first, then the call hangs). See #233.
func newToolCLIArgs(args json.RawMessage) ([]string, error) {
	var p struct {
		ObjectId    string   `json:"object_id"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
		Type        string   `json:"type"`
		Blob        string   `json:"blob"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, err
	}

	cliArgs := []string{"-edit=false"}

	if p.ObjectId != "" {
		cliArgs = append(cliArgs, "-object-id", p.ObjectId)
	}
	if p.Description != "" {
		cliArgs = append(cliArgs, "-description", p.Description)
	}
	if len(p.Tags) > 0 {
		cliArgs = append(cliArgs, "-tags", strings.Join(p.Tags, ","))
	}
	if p.Type != "" {
		cliArgs = append(cliArgs, "-type", p.Type)
	}
	if p.Blob != "" {
		cliArgs = append(cliArgs, "-blob", p.Blob)
	}

	return cliArgs, nil
}

// importToolCLIArgs translates the `import` tool's arguments into `dodder
// import` CLI args: -omit-tags / -blob_store-id flags first, then the
// variadic inventory-list paths.
func importToolCLIArgs(args json.RawMessage) ([]string, error) {
	var p struct {
		Paths                 []string          `json:"paths"`
		BlobStoreId           string            `json:"blob_store_id"`
		OmitTags              []string          `json:"omit_tags"`
		DryRun                bool              `json:"dry_run"`
		PlanFormat            string            `json:"plan_format"`
		BloblessTypeRemapping map[string]string `json:"blobless_type_remapping"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, err
	}

	var cliArgs []string
	if p.DryRun {
		cliArgs = append(cliArgs, "-dry-run")
	}
	if p.PlanFormat != "" {
		cliArgs = append(cliArgs, "-plan-format", p.PlanFormat)
	}
	for _, pattern := range p.OmitTags {
		cliArgs = append(cliArgs, "-omit-tags", pattern)
	}
	for old, replacement := range p.BloblessTypeRemapping {
		cliArgs = append(cliArgs, "-resolve-blobless-type", old+"="+replacement)
	}
	if p.BlobStoreId != "" {
		cliArgs = append(cliArgs, "-blob_store-id", p.BlobStoreId)
	}
	cliArgs = append(cliArgs, p.Paths...)

	return cliArgs, nil
}

// pushPullToolCLIArgs translates the shared push/pull tool arguments into
// CLI args: the -direct flag first, then the single remote repo-id
// positional, then query terms. remote/direct are omitted when empty so
// the command falls back to the workspace's pinned parent (#287b).
func pushPullToolCLIArgs(args json.RawMessage) ([]string, error) {
	var p struct {
		Remote string   `json:"remote"`
		Query  []string `json:"query"`
		Direct string   `json:"direct"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, err
	}

	var cliArgs []string
	if p.Direct != "" {
		cliArgs = append(cliArgs, "-direct", p.Direct)
	}
	if p.Remote != "" {
		cliArgs = append(cliArgs, p.Remote)
	}
	cliArgs = append(cliArgs, p.Query...)

	return cliArgs, nil
}

func makeEditHandler(
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
			RepoId      string   `json:"repo_id"`
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

		// Open the addressed repo per call (FDR-0019 #278) and mutate within
		// its context lifecycle; empty repo_id resolves to the server's
		// default.
		if err := bridge.WithRepo(
			ctx,
			p.RepoId,
			func(repo *local_working_copy.Repo) error {
				op := repo_actions.MakeUpdateObject(repo)

				changes := repo_actions.ObjectChanges{
					Description: p.Description,
					Tags:        p.Tags,
					Type:        p.Type,
					Blob:        p.Blob,
				}

				_, err := op.Run(objectId, changes)
				return err
			},
		); err != nil {
			return protocol.ErrorResultV1(formatToolError(err)), nil
		}

		// Show the updated object from the same repo (after the edit flush).
		result, err := bridge.RunCommandWithRepoId(ctx, "show", []string{p.ObjectId}, defaultMaxBytes, p.RepoId)
		if err != nil {
			return protocol.ErrorResultV1(formatToolError(err)), nil
		}

		return &protocol.ToolCallResultV1{
			Content: []protocol.ContentBlockV1{
				protocol.TextContentV1(result.Stdout),
			},
		}, nil
	}
}

// organizeQueryGroup builds the query group both organize tools use, matching
// the CLI organize command's options (default genre zettels, latest sigil,
// non-empty query required). orgie-extract: the organize_plan/organize_commit
// MCP handlers and the CLI share repo_actions.OrganizePlan; this is the MCP
// query-building counterpart of organize.go's MakeQueryResolvingFilenames.
func organizeQueryGroup(
	repo *local_working_copy.Repo,
	query []string,
) (*queries.Query, error) {
	return repo.MakeExternalQueryGroup(
		queries.BuilderOptions(
			queries.BuilderOptionRequireNonEmptyQuery(),
			queries.BuilderOptionDefaultGenres(genres.Zettel),
			queries.BuilderOptionDefaultSigil(ids.SigilLatest),
		),
		sku.ExternalQueryOptions{},
		query...,
	)
}

func makeOrganizePlanHandler(bridge Bridge) server.ToolHandlerV1 {
	return func(
		ctx context.Context,
		args json.RawMessage,
	) (*protocol.ToolCallResultV1, error) {
		var p struct {
			Query  []string `json:"query"`
			RepoId string   `json:"repo_id"`
		}

		if err := json.Unmarshal(args, &p); err != nil {
			return protocol.ErrorResultV1(
				fmt.Sprintf("Invalid arguments: %v", err),
			), nil
		}

		var buf strings.Builder

		if err := bridge.WithRepo(
			ctx,
			p.RepoId,
			func(repo *local_working_copy.Repo) error {
				queryGroup, err := organizeQueryGroup(repo, p.Query)
				if err != nil {
					return err
				}

				before, _, _, err := repo_actions.OrganizePlan(
					repo,
					queryGroup,
					orgie.MakeFlags(),
				)
				if err != nil {
					return err
				}

				_, err = before.WriteTo(&buf)
				return err
			},
		); err != nil {
			return protocol.ErrorResultV1(formatToolError(err)), nil
		}

		return &protocol.ToolCallResultV1{
			Content: []protocol.ContentBlockV1{
				protocol.TextContentV1(buf.String()),
			},
		}, nil
	}
}

func makeOrganizeCommitHandler(bridge Bridge) server.ToolHandlerV1 {
	return func(
		ctx context.Context,
		args json.RawMessage,
	) (*protocol.ToolCallResultV1, error) {
		var p struct {
			Query    []string `json:"query"`
			Organize string   `json:"organize"`
			RepoId   string   `json:"repo_id"`
		}

		if err := json.Unmarshal(args, &p); err != nil {
			return protocol.ErrorResultV1(
				fmt.Sprintf("Invalid arguments: %v", err),
			), nil
		}

		var summary string

		if err := bridge.WithRepo(
			ctx,
			p.RepoId,
			func(repo *local_working_copy.Repo) error {
				queryGroup, err := organizeQueryGroup(repo, p.Query)
				if err != nil {
					return err
				}

				// Re-derive the before-edit baseline statelessly from the same
				// query, then diff the submitted (edited) buffer against it.
				// Use the effective query OrganizePlan returns (after default-
				// query substitution) so the commit diffs against the same
				// baseline the plan rendered.
				before, original, effective, err := repo_actions.OrganizePlan(
					repo,
					queryGroup,
					orgie.MakeFlags(),
				)
				if err != nil {
					return err
				}

				changes, err := repo_actions.OrganizeCommitFromReader(
					repo,
					effective,
					before,
					original,
					strings.NewReader(p.Organize),
				)
				if err != nil {
					return err
				}

				summary = fmt.Sprintf(
					"organize committed: %d changed, %d added, %d removed",
					changes.Changed.Len(),
					changes.Added.Len(),
					changes.Removed.Len(),
				)

				return nil
			},
		); err != nil {
			return protocol.ErrorResultV1(formatToolError(err)), nil
		}

		return &protocol.ToolCallResultV1{
			Content: []protocol.ContentBlockV1{
				protocol.TextContentV1(summary),
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

// emptyOutputMessage is the placeholder text emitted when a bridge
// produces no stdout (and no stderr / truncation suffix). MCP clients
// (e.g. Claude Code's zod-based validator) reject content blocks whose
// `text` field is the empty string with an `invalid_union` error.
// Filling the block with a stable, predictable message keeps the
// response well-formed and tells the model "the command ran but said
// nothing." See amarbel-llc/dodder#213.
const emptyOutputMessage = "no output"

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

		// repo_id is an optional param common to the bridge-routed tools;
		// extract it here and run the command against that repo (FDR-0019).
		// The tool-specific translate funcs ignore it — json.Unmarshal drops
		// unknown fields — so each tool need only add it to its schema.
		var rp struct {
			RepoId string `json:"repo_id"`
		}
		_ = json.Unmarshal(args, &rp)

		result, err := bridge.RunCommandWithRepoId(
			ctx,
			cmdName,
			cliArgs,
			defaultMaxBytes,
			rp.RepoId,
		)
		if err != nil {
			errMsg := formatToolError(err)
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

		if output == "" {
			output = emptyOutputMessage
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
			errMsg := formatToolError(err)
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

		if output == "" {
			output = emptyOutputMessage
		}

		return &protocol.ToolCallResultV1{
			Content: []protocol.ContentBlockV1{
				protocol.TextContentV1(output),
			},
		}, nil
	}
}
