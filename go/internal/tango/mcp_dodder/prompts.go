package mcp_dodder

import (
	"context"
	"fmt"
	"strings"

	"github.com/amarbel-llc/purse-first/libs/go-mcp/protocol"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/server"
)

func registerPrompts(prompts *server.PromptRegistry) {
	prompts.Register(
		protocol.Prompt{
			Name:        "discover",
			Description: "Step-by-step recipe to discover what's in a dodder repo: browse the type/tag word indexes, search by word, and drill into facets.",
			Arguments: []protocol.PromptArgument{
				{
					Name:        "word",
					Description: "A word to search for in tag and type names (e.g. 'project', 'task', 'area'). Omit to start with the full word indexes.",
				},
			},
		},
		renderDiscover,
	)

	prompts.Register(
		protocol.Prompt{
			Name:        "query-objects",
			Description: "Step-by-step recipe to find dodder objects using AND-combined query filters (type, tag, genre).",
			Arguments: []protocol.PromptArgument{
				{
					Name:        "type",
					Description: "Type filter (e.g. task, md). Omit to match all types.",
				},
				{
					Name:        "tag",
					Description: "Tag filter (e.g. todo, area-home). Omit to match all tags.",
				},
				{
					Name:        "genre",
					Description: "Genre filter: :z (zettels), :e (tags), :t (types). Omit to match all genres.",
				},
			},
		},
		renderQueryObjects,
	)

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
}

func renderDiscover(
	ctx context.Context,
	args map[string]string,
) (*protocol.PromptGetResult, error) {
	word := args["word"]

	if word != "" {
		return &protocol.PromptGetResult{
			Description: fmt.Sprintf("Discover types and tags matching '%s'", word),
			Messages: []protocol.PromptMessage{
				{
					Role: "user",
					Content: protocol.TextContent(fmt.Sprintf(`## Goal: Discover what's in this dodder repo related to "%s"

### Step 1: Search for matching types
Call query-type with words: ["%s"]
This returns type summaries matching the word. Each result includes
object-id, description, tags (meta-tags), and a resource-uri.

### Step 2: Search for matching tags
Call query-tag with words: ["%s"]
This returns tag summaries matching the word. Each result includes
object-id, description, its own tags (meta-tags), and a resource-uri.

### Step 3: Filter results using meta-tags
Each result has a "tags" field containing meta-tags that describe the
type or tag itself. Use meta-tags to filter — e.g. check for specific
meta-tags that indicate status, priority, or categorization.

### Step 4: Drill into interesting results
For types, read the facets resource to understand the data shape:
  dodder://types/<type-id>/objects/facets

For tags, read the facets resource to see what's tagged:
  dodder://tags/<tag-id>/objects/facets

Facets show object counts grouped by tag prefix, revealing the
tag taxonomy used in this repo.

### Step 5: Browse objects (optional)
  dodder://types/<type-id>/objects — all objects of a type (box format)
  dodder://tags/<tag-id>/objects — all objects with a tag (box format)`, word, word, word)),
				},
			},
		}, nil
	}

	return &protocol.PromptGetResult{
		Description: "Discover what's in this dodder repo",
		Messages: []protocol.PromptMessage{
			{
				Role: "user",
				Content: protocol.TextContent(`## Goal: Discover what's in this dodder repo

### Step 1: Browse the word indexes
Read these resources to see what words appear in type and tag names:
  dodder://types_index — words from all type names, with counts
  dodder://tags_index — words from all tag names, with counts

These give you the vocabulary of this repo without loading all objects.

### Step 2: Search by word
Pick interesting words from the indexes and search:
  Call query-type with words: ["<word>"] — finds matching types
  Call query-tag with words: ["<word>"] — finds matching tags

Results include meta-tags in the "tags" field that describe each type/tag.

### Step 3: Drill into interesting types or tags
For types, read the facets resource to understand the data shape:
  dodder://types/<type-id>/objects/facets

For tags, read the facets resource to see what's tagged:
  dodder://tags/<tag-id>/objects/facets

Facets show object counts grouped by tag prefix, revealing the
tag taxonomy used in this repo.

### Step 4: Browse objects (optional)
  dodder://types/<type-id>/objects — all objects of a type (box format)
  dodder://tags/<tag-id>/objects — all objects with a tag (box format)`),
			},
		},
	}, nil
}

func renderQueryObjects(
	ctx context.Context,
	args map[string]string,
) (*protocol.PromptGetResult, error) {
	typeName := args["type"]
	tag := args["tag"]
	genre := args["genre"]

	var queryTerms []string
	if genre != "" {
		queryTerms = append(queryTerms, fmt.Sprintf("%q", genre))
	}
	if typeName != "" {
		queryTerms = append(queryTerms, fmt.Sprintf("\"!%s\"", typeName))
	}
	if tag != "" {
		queryTerms = append(queryTerms, fmt.Sprintf("%q", tag))
	}

	queryArray := "[" + strings.Join(queryTerms, ", ") + "]"
	if len(queryTerms) == 0 {
		queryArray = `["<term1>", "<term2>"]`
	}

	return &protocol.PromptGetResult{
		Description: "Find objects using AND-combined query filters",
		Messages: []protocol.PromptMessage{
			{
				Role: "user",
				Content: protocol.TextContent(fmt.Sprintf(`## Goal: Query for dodder objects

### Step 1: Run the query
Call query with:
  query: %s
  format: "box"
  limit: 50

Query terms are AND-combined — results must match ALL terms.
Term types:
  - Genre filters: :z (zettels), :e (tags), :t (types)
  - Type filters: !<type-name> (e.g. !task, !md)
  - Tag filters: bare tag name (e.g. todo, area-home)

### Step 2: Read box-format results
Each line in the output looks like:
  [<object-id> @<blob-digest> !<type> <tag1> <tag2> ...] <description>

The description after the closing bracket summarizes the object.
Tags inside brackets show the object's full tag set.

### Step 3: Inspect individual objects (optional)
To see an object's full metadata and traversal links, read:
  dodder://objects/<object-id>

If the response includes "blob-formats-resource", follow it to
discover available formatters and read the blob content.

### Step 4: Refine the query (optional)
Add more terms to narrow results. Examples:
  [":z", "!md"] — all markdown zettels
  ["!task", "todo"] — tasks tagged with todo
  [":e", "area"] — tags in the "area" namespace`, queryArray)),
			},
		},
	}, nil
}

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
Call query-type with words: ["%s"]
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
Call query-tag with words: ["%s"]
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
