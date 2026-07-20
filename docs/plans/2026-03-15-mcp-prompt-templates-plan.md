# MCP Prompt Templates Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add 5 workflow prompt templates to the dodder MCP server so agents can follow step-by-step recipes for progressive disclosure.

**Architecture:** Register prompts via `server.NewPromptRegistry()`. Each prompt is a pure string renderer (no bridge calls) that returns a markdown recipe with exact tool calls and resource URIs. One new file `prompts.go` for all prompt logic, one 3-line change to `server.go` to wire it up.

**Tech Stack:** go-mcp `server.PromptRegistry`, `protocol.Prompt`, `protocol.PromptGetResult`, `protocol.PromptMessage`, `protocol.TextContent`

**Rollback:** Remove `Prompts: prompts` from `server.Options` in `server.go`. N/A — purely additive.

---

### Task 1: Create prompts.go with registerPrompts and summarize-projects prompt

**Promotion criteria:** N/A

**Files:**
- Create: `go/internal/tango/mcp_dodder/prompts.go`

**Step 1: Create prompts.go with registerPrompts and the first prompt**

```go
package mcp_dodder

import (
	"context"
	"fmt"
	"strings"

	"code.linenisgreat.com/purse-first/libs/go-mcp/protocol"
	"code.linenisgreat.com/purse-first/libs/go-mcp/server"
)

func registerPrompts(prompts *server.PromptRegistry) {
	prompts.Register(
		protocol.Prompt{
			Name:        "summarize-projects",
			Description: "Step-by-step recipe to find and summarize all active projects with their tag distributions.",
		},
		renderSummarizeProjects,
	)
}

func renderSummarizeProjects(
	ctx context.Context,
	args map[string]string,
) (*protocol.PromptGetResult, error) {
	return &protocol.PromptGetResult{
		Description: "Find active projects and summarize their contents",
		Messages: []protocol.PromptMessage{
			{
				Role: "user",
				Content: protocol.TextContent(`## Goal: Summarize active projects

### Step 1: Find project tags
Call dodder_tag_query with words: ["project"]
This returns all tags containing "project" in their name (e.g. project-2024-q3, project-dodder).

### Step 2: Filter for active projects
In the results array, each tag has a "tags" field listing its meta-tags.
Keep only tags where the "tags" array contains "active".
Tags without "active" in their meta-tags are inactive/archived projects.

### Step 3: Get analytics for each active project
For each active project tag, read the resource:
  dodder://tags/<tag-id>/objects/facets

Replace <tag-id> with the tag's object-id from step 2.
This returns a breakdown of object counts grouped by tag prefix (e.g. priority-, urgency-, area-).

### Step 4: Optionally browse project objects
To see what objects belong to a project, read:
  dodder://tags/<tag-id>/objects

This returns a box-format listing (one line per object) with object IDs, types, tags, and descriptions.

### Step 5: Summarize
Compile the facets from step 3 into a summary table showing each active project's name, total object count, and notable tag distributions (priorities, types, areas).`),
			},
		},
	}, nil
}
```

**Step 2: Verify it compiles**

Run: `just build` (from repo root)
Expected: Successful build (prompts.go compiles but isn't wired up yet)

**Step 3: Commit**

```bash
git add go/internal/tango/mcp_dodder/prompts.go
git commit -m "feat: add prompts.go with summarize-projects workflow prompt"
```

---

### Task 2: Add find-tasks prompt

**Promotion criteria:** N/A

**Files:**
- Modify: `go/internal/tango/mcp_dodder/prompts.go`

**Step 1: Register find-tasks in registerPrompts and add renderer**

Add to `registerPrompts`:
```go
	prompts.Register(
		protocol.Prompt{
			Name:        "find-tasks",
			Description: "Step-by-step recipe to find tasks, optionally filtered by priority or tag.",
			Arguments: []protocol.PromptArgument{
				{
					Name:        "priority",
					Description: "Priority level filter (e.g. priority-0_must, priority-1_should, priority-2_want). Omit for all priorities.",
				},
				{
					Name:        "tag",
					Description: "Additional tag filter (e.g. area-home, project-dodder). Omit for all tags.",
				},
			},
		},
		renderFindTasks,
	)
```

Add renderer function:
```go
func renderFindTasks(
	ctx context.Context,
	args map[string]string,
) (*protocol.PromptGetResult, error) {
	priority := args["priority"]
	tag := args["tag"]

	var queryTerms []string
	queryTerms = append(queryTerms, `"!task"`)
	if priority != "" {
		queryTerms = append(queryTerms, fmt.Sprintf("%q", priority))
	}
	if tag != "" {
		queryTerms = append(queryTerms, fmt.Sprintf("%q", tag))
	}
	queryArray := "[" + strings.Join(queryTerms, ", ") + "]"

	return &protocol.PromptGetResult{
		Description: "Find tasks with optional priority and tag filters",
		Messages: []protocol.PromptMessage{
			{
				Role: "user",
				Content: protocol.TextContent(fmt.Sprintf(`## Goal: Find tasks

### Step 1: Query for tasks
Call dodder_query with:
  query: %s
  format: "box"
  limit: 50

Query terms are AND-combined, so this returns objects that match ALL terms.
The "box" format gives a compact one-line-per-object listing.

### Step 2: Examine results
Each line in box format looks like:
  [<object-id> @<blob-digest> !task <tag1> <tag2> ...] <description>

The description after the closing bracket tells you what the task is about.
Tags inside the brackets show priority, urgency, area, and project associations.

### Step 3: Read task content (optional)
To see a task's full content, read the resource:
  dodder://objects/<object-id>

This returns metadata with traversal links. If the task has blob content,
follow the blob-formats-resource link to discover formatters, then read
the formatted blob.

### Step 4: Get task distribution overview (optional)
To see how tasks break down by tag, read:
  dodder://types/task/objects/facets

This shows counts grouped by tag prefix (priority, urgency, area, project).`, queryArray)),
			},
		},
	}, nil
}
```

**Step 2: Verify it compiles**

Run: `just build`
Expected: Successful build

**Step 3: Commit**

```bash
git add go/internal/tango/mcp_dodder/prompts.go
git commit -m "feat: add find-tasks workflow prompt with priority/tag filters"
```

---

### Task 3: Add read-object prompt

**Promotion criteria:** N/A

**Files:**
- Modify: `go/internal/tango/mcp_dodder/prompts.go`

**Step 1: Register read-object in registerPrompts and add renderer**

Add to `registerPrompts`:
```go
	prompts.Register(
		protocol.Prompt{
			Name:        "read-object",
			Description: "Step-by-step recipe to inspect a dodder object and read its blob content through format discovery.",
			Arguments: []protocol.PromptArgument{
				{
					Name:        "object_id",
					Description: "Object identifier (e.g. zettel ID like 'ceroplastes/midtown', tag like 'todo', or type like '!md').",
					Required:    true,
				},
			},
		},
		renderReadObject,
	)
```

Add renderer function:
```go
func renderReadObject(
	ctx context.Context,
	args map[string]string,
) (*protocol.PromptGetResult, error) {
	objectId := args["object_id"]
	if objectId == "" {
		objectId = "<object_id>"
	}

	return &protocol.PromptGetResult{
		Description: "Inspect an object and read its blob content",
		Messages: []protocol.PromptMessage{
			{
				Role: "user",
				Content: protocol.TextContent(fmt.Sprintf(`## Goal: Read object content for %s

### Step 1: Get object metadata
Read the resource:
  dodder://objects/%s

This returns JSON with: object-id, date, description, type, tags, and
traversal links to related resources.

### Step 2: Discover blob formatters
If the response includes a "blob-formats-resource" field, read that URI:
  dodder://objects/%s/blob/formats

This returns a list of available formatter IDs with their resource URIs.
If there is no "blob-formats-resource" field, the object has no blob content — skip to step 4.

### Step 3: Read formatted blob content
Pick a formatter from step 2 and read its resource URI:
  dodder://objects/%s/blob/formats/<format-id>

This renders the blob content using that formatter. Common formatters
depend on the object's type (e.g. markdown types may have text formatters).

### Step 4: Explore related objects (optional)
The step 1 response includes traversal links:
- "type-resource" → the type definition for this object
- "type-objects-resource" → all objects sharing this type
- "tag-resources" → each tag with links to its objects
- "markl-resource" → integrity/provenance data (rarely needed)`, objectId, objectId, objectId, objectId)),
			},
		},
	}, nil
}
```

**Step 2: Verify it compiles**

Run: `just build`
Expected: Successful build

**Step 3: Commit**

```bash
git add go/internal/tango/mcp_dodder/prompts.go
git commit -m "feat: add read-object workflow prompt with blob format discovery"
```

---

### Task 4: Add explore-type and explore-tag prompts

**Promotion criteria:** N/A

**Files:**
- Modify: `go/internal/tango/mcp_dodder/prompts.go`

**Step 1: Register both prompts in registerPrompts and add renderers**

Add to `registerPrompts`:
```go
	prompts.Register(
		protocol.Prompt{
			Name:        "explore-type",
			Description: "Step-by-step recipe to explore a dodder type: find it, view its metadata, analyze tag distribution, and browse its objects.",
			Arguments: []protocol.PromptArgument{
				{
					Name:        "type",
					Description: "Type name to explore (e.g. task, md, toml-type-v1). Do not include the ! prefix.",
					Required:    true,
				},
			},
		},
		renderExploreType,
	)

	prompts.Register(
		protocol.Prompt{
			Name:        "explore-tag",
			Description: "Step-by-step recipe to explore a dodder tag: find it, view its meta-tags, analyze tagged objects, and browse contents.",
			Arguments: []protocol.PromptArgument{
				{
					Name:        "tag",
					Description: "Tag name to explore (e.g. todo, project-dodder, area-home).",
					Required:    true,
				},
			},
		},
		renderExploreTag,
	)
```

Add renderer functions:
```go
func renderExploreType(
	ctx context.Context,
	args map[string]string,
) (*protocol.PromptGetResult, error) {
	typeName := args["type"]
	if typeName == "" {
		typeName = "<type>"
	}

	return &protocol.PromptGetResult{
		Description: fmt.Sprintf("Explore the %s type", typeName),
		Messages: []protocol.PromptMessage{
			{
				Role: "user",
				Content: protocol.TextContent(fmt.Sprintf(`## Goal: Explore the %s type

### Step 1: Find the type
Call dodder_type_query with words: ["%s"]
This returns type summaries matching the word. Each result includes
object-id, description, tags, and a resource-uri for drill-down.

### Step 2: Get type metadata
Read the resource:
  dodder://types/%s

This returns the type's metadata with links to sub-resources:
blob-resource, objects-resource, and markl-resource.

### Step 3: Analyze tag distribution
Read the resource:
  dodder://types/%s/objects/facets

This shows how objects of this type are distributed across tags,
grouped by tag prefix (priority-, urgency-, area-, project-).
Use this to understand the shape of the data.

### Step 4: Browse objects
Read the resource:
  dodder://types/%s/objects

This returns a box-format listing of all objects with this type.
Each line shows: [object-id @blob-digest !type tag1 tag2 ...] description

### Step 5: Inspect individual objects (optional)
Pick an object-id from step 4 and read:
  dodder://objects/<object-id>

Follow the read-object workflow to see its content.`, typeName, typeName, typeName, typeName, typeName)),
			},
		},
	}, nil
}

func renderExploreTag(
	ctx context.Context,
	args map[string]string,
) (*protocol.PromptGetResult, error) {
	tag := args["tag"]
	if tag == "" {
		tag = "<tag>"
	}

	return &protocol.PromptGetResult{
		Description: fmt.Sprintf("Explore the %s tag", tag),
		Messages: []protocol.PromptMessage{
			{
				Role: "user",
				Content: protocol.TextContent(fmt.Sprintf(`## Goal: Explore the %s tag

### Step 1: Find the tag
Call dodder_tag_query with words: ["%s"]
This returns tag summaries matching the word. Each result includes
object-id, description, its own tags (meta-tags), and a resource-uri.

### Step 2: Check meta-tags
Look at the "tags" field in the result. Meta-tags tell you about the
tag itself — e.g. "active" means currently active, "priority-0_must"
means high priority, "area-home" is the life area.

### Step 3: Get tag metadata
Read the resource:
  dodder://tags/%s

This returns the tag's full metadata with links to objects-resource
and markl-resource.

### Step 4: Analyze tagged objects
Read the resource:
  dodder://tags/%s/objects/facets

This shows how objects with this tag break down by other tags,
grouped by prefix. Useful for understanding what kinds of content
carry this tag.

### Step 5: Browse tagged objects
Read the resource:
  dodder://tags/%s/objects

This returns a box-format listing of all objects with this tag.

### Step 6: Inspect individual objects (optional)
Pick an object-id from step 5 and read:
  dodder://objects/<object-id>

Follow the read-object workflow to see its content.`, tag, tag, tag, tag, tag)),
			},
		},
	}, nil
}
```

**Step 2: Verify it compiles**

Run: `just build`
Expected: Successful build

**Step 3: Commit**

```bash
git add go/internal/tango/mcp_dodder/prompts.go
git commit -m "feat: add explore-type and explore-tag workflow prompts"
```

---

### Task 5: Wire prompts into server.go

**Promotion criteria:** N/A

**Files:**
- Modify: `go/internal/tango/mcp_dodder/server.go:145-172`

**Step 1: Add prompt registry to RunServer**

In `RunServer`, after line 162 (`registerResources(resources, index, tagIdx, bridge)`), add:

```go
	prompts := server.NewPromptRegistry()
	registerPrompts(prompts)
```

Then in the `server.Options` struct (line 166-172), add `Prompts: prompts`:

```go
	srv, err := server.New(t, server.Options{
		ServerName:    "dodder",
		ServerVersion: "0.1.0",
		Tools:         tools,
		Resources:     provider,
		Prompts:       prompts,
		Instructions:  mcpInstructions,
	})
```

**Step 2: Verify it compiles and tests pass**

Run: `just build`
Expected: Successful build

Run: `just test`
Expected: All tests pass

**Step 3: Commit**

```bash
git add go/internal/tango/mcp_dodder/server.go
git commit -m "feat: wire prompt templates into dodder MCP server"
```

---

### Task 6: Build, install, and verify prompts work end-to-end

**Promotion criteria:** N/A

**Files:** None (verification only)

**Step 1: Build and install**

Run: `just build` (from repo root)
Run: `just install-dodder` (from ~/eng, installs MCP server)

**Step 2: Restart Claude Code and verify prompts are listed**

After restart, the dodder MCP server should advertise 5 prompts via `prompts/list`.
Use a subagent to exercise the `summarize-projects` workflow by following the
prompt template steps.

**Step 3: Verify each prompt renders correctly**

Test each prompt with the MCP client to confirm they return well-formed
markdown recipes with correct tool names and resource URIs.
