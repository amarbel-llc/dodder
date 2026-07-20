# MCP Type Resources & type_query Tool Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Add progressive disclosure for dodder types via MCP resources (`dodder://types_index`, `dodder://types`, `dodder://types/<id>`, `dodder://types/<id>/blob`) and a `type_query` tool with word-based OR-union search.

**Architecture:** New `type_index.go` builds an in-memory word→typeSummary index from all type objects (queried via bridge with `:t`). Word extraction uses the existing `expansion.ExpanderRight` package to break type IDs, descriptions, and tags into searchable tokens. A custom `ResourceProvider` wraps `server.ResourceRegistry` for template URI routing (same pattern as nebulous). The bridge's `RunCommand` fetches individual type objects and blobs on demand for drill-down resources.

**Tech Stack:** Go, `go-mcp` library, `expansion` package from `go/lib/charlie/expansion/`.

**Rollback:** Purely additive — revert the commit to remove.

---

### Task 1: Create typeSummary type and word index builder

**Files:**
- Create: `go/internal/hotel/mcp_dodder/type_index.go`

**Step 1: Write type_index.go**

This file defines the `typeSummary` struct (compact representation for search results), the `typeIndex` struct with word→summaries map, and the build/query logic.

```go
package mcp_dodder

import (
	"encoding/json"
	"strings"
	"sync"

	"code.linenisgreat.com/dodder/go/lib/charlie/expansion"
)

type typeSummary struct {
	ObjectId    string   `json:"object-id"`
	Date        string   `json:"date"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	ResourceURI string   `json:"resource-uri"`
}

type typeIndex struct {
	bridge Bridge
	once   sync.Once
	words  map[string][]typeSummary
	err    error
}

func makeTypeIndex(bridge Bridge) *typeIndex {
	return &typeIndex{bridge: bridge}
}

func (idx *typeIndex) ensureBuilt() error {
	idx.once.Do(func() { idx.err = idx.build() })
	return idx.err
}

func (idx *typeIndex) build() error {
	result, err := idx.bridge.RunCommand(
		nil,
		"show",
		[]string{"-format", "json", ":t"},
		500_000,
	)
	if err != nil {
		return err
	}

	idx.words = make(map[string][]typeSummary)

	for _, line := range strings.Split(result.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var obj struct {
			ObjectId    string   `json:"object-id"`
			Date        string   `json:"date"`
			Description string   `json:"description"`
			Tags        []string `json:"tags"`
		}

		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}

		summary := typeSummary{
			ObjectId:    obj.ObjectId,
			Date:        obj.Date,
			Description: obj.Description,
			Tags:        obj.Tags,
			ResourceURI: "dodder://types/" + strings.TrimPrefix(obj.ObjectId, "!"),
		}

		seen := make(map[string]bool)

		addWords := func(source string) {
			expansion.ExpanderRight.Expand(source)(func(word string) bool {
				word = strings.ToLower(word)
				if word == "" || seen[word] {
					return true
				}
				seen[word] = true
				idx.words[word] = append(idx.words[word], summary)
				return true
			})
		}

		// Expand type ID (strip ! prefix)
		addWords(strings.TrimPrefix(obj.ObjectId, "!"))

		// Expand description words
		for _, word := range strings.Fields(obj.Description) {
			addWords(strings.ToLower(word))
		}

		// Expand tags (strip % prefix)
		for _, tag := range obj.Tags {
			addWords(strings.TrimPrefix(tag, "%"))
		}
	}

	return nil
}

func (idx *typeIndex) sortedWords() []string {
	words := make([]string, 0, len(idx.words))
	for w := range idx.words {
		words = append(words, w)
	}
	sort.Strings(words)
	return words
}

func (idx *typeIndex) query(queryWords []string) []typeSummary {
	seen := make(map[string]bool)
	var results []typeSummary

	for _, qw := range queryWords {
		qw = strings.ToLower(qw)
		for _, summary := range idx.words[qw] {
			if !seen[summary.ObjectId] {
				seen[summary.ObjectId] = true
				results = append(results, summary)
			}
		}
	}

	return results
}
```

Note: Add `"sort"` to the imports.

**Step 2: Build to verify compilation**

Run: `just build`
Expected: PASS

**Step 3: Commit**

```
feat: add type index with word expansion for MCP progressive disclosure
```

---

### Task 2: Create resource provider with template routing

**Files:**
- Create: `go/internal/hotel/mcp_dodder/resources.go`

**Step 1: Write resources.go**

Custom `ResourceProvider` wrapping `server.ResourceRegistry` with prefix-based routing for template URIs. Same pattern as nebulous's `feedResourceProvider`.

```go
package mcp_dodder

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"code.linenisgreat.com/purse-first/libs/go-mcp/protocol"
	"code.linenisgreat.com/purse-first/libs/go-mcp/server"
)

type typeResourceProvider struct {
	registry *server.ResourceRegistry
	index    *typeIndex
	bridge   Bridge
}

func (p *typeResourceProvider) ListResources(
	ctx context.Context,
) ([]protocol.Resource, error) {
	return p.registry.ListResources(ctx)
}

func (p *typeResourceProvider) ListResourceTemplates(
	ctx context.Context,
) ([]protocol.ResourceTemplate, error) {
	return p.registry.ListResourceTemplates(ctx)
}

func (p *typeResourceProvider) ReadResource(
	ctx context.Context,
	uri string,
) (*protocol.ResourceReadResult, error) {
	switch {
	case strings.HasPrefix(uri, "dodder://types/") &&
		strings.HasSuffix(uri, "/blob"):
		id := strings.TrimPrefix(uri, "dodder://types/")
		id = strings.TrimSuffix(id, "/blob")
		return p.readTypeBlob(ctx, id)

	case strings.HasPrefix(uri, "dodder://types/"):
		id := strings.TrimPrefix(uri, "dodder://types/")
		return p.readType(ctx, id)
	}

	return p.registry.ReadResource(ctx, uri)
}

func (p *typeResourceProvider) readType(
	ctx context.Context,
	id string,
) (*protocol.ResourceReadResult, error) {
	result, err := p.bridge.RunCommand(
		ctx,
		"show",
		[]string{"-format", "json", "!" + id},
		defaultMaxBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("show type %s: %w", id, err)
	}

	// Parse and re-emit with blob resource link
	var obj map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(result.Stdout)), &obj); err != nil {
		return nil, fmt.Errorf("parse type %s: %w", id, err)
	}
	delete(obj, "blob-string")
	obj["blob-resource"] = fmt.Sprintf("dodder://types/%s/blob", id)

	output, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return nil, err
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      fmt.Sprintf("dodder://types/%s", id),
			MimeType: "application/json",
			Text:     string(output),
		}},
	}, nil
}

func (p *typeResourceProvider) readTypeBlob(
	ctx context.Context,
	id string,
) (*protocol.ResourceReadResult, error) {
	result, err := p.bridge.RunCommand(
		ctx,
		"format-blob",
		[]string{"!" + id},
		defaultMaxBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("format-blob type %s: %w", id, err)
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      fmt.Sprintf("dodder://types/%s/blob", id),
			MimeType: "text/plain",
			Text:     result.Stdout,
		}},
	}, nil
}

func registerResources(
	registry *server.ResourceRegistry,
	index *typeIndex,
	bridge Bridge,
) {
	// Static: types_index — word list for discovery
	registry.RegisterResource(
		protocol.Resource{
			URI:         "dodder://types_index",
			Name:        "Type Word Index",
			Description: "Word list for type discovery. Start here, then use type_query tool or drill into dodder://types/<id>.",
			MimeType:    "application/json",
		},
		func(ctx context.Context, uri string) (*protocol.ResourceReadResult, error) {
			if err := index.ensureBuilt(); err != nil {
				return nil, err
			}

			type wordEntry struct {
				Word  string `json:"word"`
				Count int    `json:"count"`
			}

			words := index.sortedWords()
			entries := make([]wordEntry, len(words))
			for i, w := range words {
				entries[i] = wordEntry{Word: w, Count: len(index.words[w])}
			}

			result := struct {
				TotalWords int         `json:"total_words"`
				TotalTypes int         `json:"total_types"`
				Words      []wordEntry `json:"words"`
			}{
				TotalWords: len(words),
				TotalTypes: countUniqueTypes(index),
				Words:      entries,
			}

			output, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return nil, err
			}

			return &protocol.ResourceReadResult{
				Contents: []protocol.ResourceContent{{
					URI:      uri,
					MimeType: "application/json",
					Text:     string(output),
				}},
			}, nil
		},
	)

	// Static: types — list of all type resources
	registry.RegisterResource(
		protocol.Resource{
			URI:         "dodder://types",
			Name:        "All Types",
			Description: "List of all type objects with resource URIs. Use dodder://types/<id> for full metadata.",
			MimeType:    "application/json",
		},
		func(ctx context.Context, uri string) (*protocol.ResourceReadResult, error) {
			if err := index.ensureBuilt(); err != nil {
				return nil, err
			}

			seen := make(map[string]bool)
			var types []typeSummary

			for _, summaries := range index.words {
				for _, s := range summaries {
					if !seen[s.ObjectId] {
						seen[s.ObjectId] = true
						types = append(types, s)
					}
				}
			}

			output, err := json.MarshalIndent(types, "", "  ")
			if err != nil {
				return nil, err
			}

			return &protocol.ResourceReadResult{
				Contents: []protocol.ResourceContent{{
					URI:      uri,
					MimeType: "application/json",
					Text:     string(output),
				}},
			}, nil
		},
	)

	// Template: types/<id>
	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder://types/{type_id}",
			Name:        "Type Object",
			Description: "Full type metadata (without blob). Includes blob-resource link.",
			MimeType:    "application/json",
		},
		nil,
	)

	// Template: types/<id>/blob
	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder://types/{type_id}/blob",
			Name:        "Type Blob",
			Description: "Type blob content (TOML configuration).",
			MimeType:    "text/plain",
		},
		nil,
	)
}

func countUniqueTypes(index *typeIndex) int {
	seen := make(map[string]bool)
	for _, summaries := range index.words {
		for _, s := range summaries {
			seen[s.ObjectId] = true
		}
	}
	return len(seen)
}
```

**Step 2: Build to verify compilation**

Run: `just build`
Expected: PASS

**Step 3: Commit**

```
feat: add MCP resource provider for type progressive disclosure
```

---

### Task 3: Add type_query tool and wire everything into server.go

**Files:**
- Modify: `go/internal/hotel/mcp_dodder/server.go`

**Step 1: Update RunServer to create index, resource provider, and pass resources to server**

In `RunServer`, create the type index and resource provider, register resources, and pass the provider to the server options:

```go
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
	})
	if err != nil {
		return err
	}

	return srv.Run(context.Background())
}
```

**Step 2: Update registerTools signature and add type_query tool**

Change `registerTools` to accept `*typeIndex` and add the `dodder_type_query` tool:

```go
func registerTools(tools *server.ToolRegistryV1, bridge Bridge, index *typeIndex) {
	// ... existing tools unchanged ...

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
		func(ctx context.Context, args json.RawMessage) (*protocol.ToolCallResultV1, error) {
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
```

**Step 3: Build to verify compilation**

Run: `just build`
Expected: PASS

**Step 4: Commit**

```
feat: add type_query tool and wire resources into MCP server
```

---

### Task 4: Build, install, and verify with MCP tools

**Step 1: Build**

Run: `just build`
Expected: PASS

**Step 2: Verify via MCP after session restart**

Test sequence:
1. `dodder_type_query` with `["task"]` — should return compact summaries for task-related types
2. Read `dodder://types_index` resource — should return word list with counts
3. Read `dodder://types` resource — should return all type summaries
4. Read `dodder://types/task` resource — should return full type JSON with `blob-resource` link
5. Read `dodder://types/task/blob` resource — should return TOML blob content

**Step 3: Commit final state**

```
feat: MCP type progressive disclosure verified working
```
