package mcp_dodder

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/amarbel-llc/purse-first/libs/go-mcp/protocol"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/server"
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
	if err := p.index.ensureBuilt(); err != nil {
		return nil, fmt.Errorf("build type index: %w", err)
	}

	// Find the type in the index
	targetId := "!" + id
	results := p.index.query([]string{id})

	var found *typeSummary
	for i := range results {
		if results[i].ObjectId == targetId {
			found = &results[i]
			break
		}
	}

	if found == nil {
		return nil, fmt.Errorf("type not found: %s", id)
	}

	// Build full detail from summary + blob resource link
	detail := map[string]any{
		"object-id":     found.ObjectId,
		"date":          found.Date,
		"description":   found.Description,
		"tags":          found.Tags,
		"resource-uri":  found.ResourceURI,
		"blob-resource": fmt.Sprintf("dodder://types/%s/blob", id),
	}

	output, err := json.MarshalIndent(detail, "", "  ")
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
		"show",
		[]string{"-format", "json-with-blob_string", "!" + id},
		defaultMaxBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("show type blob %s: %w", id, err)
	}

	// Parse NDJSON output and extract blob-string
	for _, line := range strings.Split(result.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var obj struct {
			BlobString string `json:"blob-string"`
		}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}

		return &protocol.ResourceReadResult{
			Contents: []protocol.ResourceContent{{
				URI:      fmt.Sprintf("dodder://types/%s/blob", id),
				MimeType: "text/plain",
				Text:     obj.BlobString,
			}},
		}, nil
	}

	return nil, fmt.Errorf("type %s has no blob content", id)
}

func registerResources(
	registry *server.ResourceRegistry,
	index *typeIndex,
	bridge Bridge,
) {
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
				entries[i] = wordEntry{
					Word:  w,
					Count: len(index.words[w]),
				}
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

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder://types/{type_id}",
			Name:        "Type Object",
			Description: "Full type metadata (without blob). Includes blob-resource link.",
			MimeType:    "application/json",
		},
		nil,
	)

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
