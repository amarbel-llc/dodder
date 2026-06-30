package mcp_dodder

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/amarbel-llc/madder/go/pkgs/scoped_id"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/protocol"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/server"
)

// Prompt bodies live in the sibling prompts.tmpl (embedded below) as named
// {{define}} blocks, one per registered prompt. Keeping the wording out of
// Go keeps the recipes reviewable as plain text. The nix build only bundles
// non-Go files matched by go/gomod.nix `extras`, so the `.tmpl` pattern there
// must include this file or the embed fails under nix.
//
//go:embed prompts.tmpl
var promptsTmplText string

var promptsTmpl = template.Must(
	template.New("prompts").Parse(promptsTmplText),
)

// renderPromptBody executes the named template block from prompts.tmpl into a
// single user-role text message.
func renderPromptBody(
	name, description string,
	data any,
) (*protocol.PromptGetResult, error) {
	var b strings.Builder
	if err := promptsTmpl.ExecuteTemplate(&b, name, data); err != nil {
		return nil, err
	}

	return &protocol.PromptGetResult{
		Description: description,
		Messages: []protocol.PromptMessage{
			{
				Role:    "user",
				Content: protocol.TextContent(b.String()),
			},
		},
	}, nil
}

func registerPrompts(
	prompts *server.PromptRegistry,
	provider *typeResourceProvider,
	startupRepoId scoped_id.Id,
	hasWorkspace bool,
	dodderVersion string,
) {
	// clown dynamic system-prompt contribution (RFC-0002 §5, dodder#277):
	// clown-stdio-bridge fetches /clown/system-prompt by issuing prompts/get
	// for the well-known name "system-prompt-append" at session launch and
	// appends the result to the agent's system prompt.
	prompts.Register(
		protocol.Prompt{
			Name:        "system-prompt-append",
			Description: "Live orientation for the bound dodder repo (clown dynamic system-prompt fragment).",
		},
		renderSystemPromptAppend(provider, startupRepoId, hasWorkspace, dodderVersion),
	)

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

// systemPromptAppendData is the template input for the clown dynamic
// system-prompt fragment (dodder#277).
type systemPromptAppendData struct {
	Version         string
	BoundRepo       string
	Scope           string
	HasWorkspace    bool
	CountsAvailable bool
	TypeCount       int
	TagCount        int
	RepoCount       int
	Repos           []string
}

// renderSystemPromptAppend builds the clown dynamic system-prompt fragment
// (dodder#277): a concise, live orientation for the repo this MCP server is
// bound to, plus the repos addressable from here. It is best-effort — a
// fresh or unreadable store must never turn this optional fragment into an
// error, so index/repo-scan failures degrade to omitted lines and the call
// still returns a fragment.
func renderSystemPromptAppend(
	provider *typeResourceProvider,
	startupRepoId scoped_id.Id,
	hasWorkspace bool,
	dodderVersion string,
) server.PromptRenderer {
	return func(
		ctx context.Context,
		args map[string]string,
	) (*protocol.PromptGetResult, error) {
		data := systemPromptAppendData{
			Version:      dodderVersion,
			BoundRepo:    repoSeg(startupRepoId),
			HasWorkspace: hasWorkspace,
		}

		if provider.startupIsCwd {
			data.Scope = "cwd-ancestor .dodder scope (spelled .name)"
		} else {
			data.Scope = "XDG-user scope (spelled name)"
		}

		// Counts are best-effort: skip silently if an index can't build.
		if idx := provider.typeIndexFor(startupRepoId); idx.ensureBuilt() == nil {
			if tagIdx := provider.tagIndexFor(startupRepoId); tagIdx.ensureBuilt() == nil {
				data.CountsAvailable = true
				data.TypeCount = countUniqueTypes(idx)
				data.TagCount = countUniqueTags(tagIdx)
			}
		}

		// Repos addressable from here (both scopes), so the agent knows
		// which -repo_id / dodder:///repos/<id> targets exist.
		if repos, err := provider.scopedRepos(); err == nil {
			data.RepoCount = len(repos)
			for _, r := range repos {
				data.Repos = append(data.Repos, r.RepoId)
			}
		}

		return renderPromptBody(
			"system-prompt-append",
			"dodder repository orientation",
			data,
		)
	}
}

type discoverData struct {
	Word string
}

func renderDiscover(
	ctx context.Context,
	args map[string]string,
) (*protocol.PromptGetResult, error) {
	word := args["word"]

	description := "Discover what's in this dodder repo"
	if word != "" {
		description = fmt.Sprintf("Discover types and tags matching '%s'", word)
	}

	return renderPromptBody("discover", description, discoverData{Word: word})
}

type queryObjectsData struct {
	QueryArray string
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

	return renderPromptBody(
		"query-objects",
		"Find objects using AND-combined query filters",
		queryObjectsData{QueryArray: queryArray},
	)
}

type readObjectData struct {
	ObjectId string
}

func renderReadObject(
	ctx context.Context,
	args map[string]string,
) (*protocol.PromptGetResult, error) {
	objectId := args["object_id"]
	if objectId == "" {
		objectId = "<object_id>"
	}

	return renderPromptBody(
		"read-object",
		"Inspect an object and read its blob content",
		readObjectData{ObjectId: objectId},
	)
}

type exploreTypeData struct {
	Type string
}

func renderExploreType(
	ctx context.Context,
	args map[string]string,
) (*protocol.PromptGetResult, error) {
	typeName := args["type"]
	if typeName == "" {
		typeName = "<type>"
	}

	return renderPromptBody(
		"explore-type",
		fmt.Sprintf("Explore the %s type", typeName),
		exploreTypeData{Type: typeName},
	)
}

type exploreTagData struct {
	Tag string
}

func renderExploreTag(
	ctx context.Context,
	args map[string]string,
) (*protocol.PromptGetResult, error) {
	tag := args["tag"]
	if tag == "" {
		tag = "<tag>"
	}

	return renderPromptBody(
		"explore-tag",
		fmt.Sprintf("Explore the %s tag", tag),
		exploreTagData{Tag: tag},
	)
}
