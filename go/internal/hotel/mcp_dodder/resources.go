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
	case strings.HasPrefix(uri, "dodder://objects/"):
		rest := strings.TrimPrefix(uri, "dodder://objects/")

		if idx := strings.Index(rest, "/blob/"); idx >= 0 {
			objectId := rest[:idx]
			format := rest[idx+len("/blob/"):]
			return p.readObjectBlob(ctx, objectId, format)
		}

		if idx := strings.LastIndex(rest, "/markl"); idx >= 0 &&
			idx+len("/markl") == len(rest) {
			objectId := rest[:idx]
			return p.readObjectMarkl(ctx, objectId)
		}

	case strings.HasPrefix(uri, "dodder://types/"):
		rest := strings.TrimPrefix(uri, "dodder://types/")

		if strings.HasSuffix(rest, "/objects") {
			id := strings.TrimSuffix(rest, "/objects")
			return p.readTypeObjects(ctx, id)
		}

		if strings.HasSuffix(rest, "/markl") {
			id := strings.TrimSuffix(rest, "/markl")
			return p.readTypeMarkl(ctx, id)
		}

		if idx := strings.Index(rest, "/blob/"); idx >= 0 {
			id := rest[:idx]
			format := rest[idx+len("/blob/"):]
			return p.readTypeBlobFormatted(ctx, id, format)
		}

		if strings.HasSuffix(rest, "/blob") {
			id := strings.TrimSuffix(rest, "/blob")
			return p.readTypeBlob(ctx, id)
		}

		return p.readType(ctx, rest)
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

	detail := map[string]any{
		"object-id":        found.ObjectId,
		"date":             found.Date,
		"description":      found.Description,
		"tags":             found.Tags,
		"resource-uri":     found.ResourceURI,
		"blob-resource":    fmt.Sprintf("dodder://types/%s/blob", id),
		"objects-resource": fmt.Sprintf("dodder://types/%s/objects", id),
		"markl-resource":   fmt.Sprintf("dodder://types/%s/markl", id),
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

func (p *typeResourceProvider) readTypeObjects(
	ctx context.Context,
	id string,
) (*protocol.ResourceReadResult, error) {
	result, err := p.bridge.RunCommand(
		ctx,
		"show",
		[]string{"-format", "box", "!" + id},
		500_000,
	)
	if err != nil {
		return nil, fmt.Errorf("query type objects %s: %w", id, err)
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      fmt.Sprintf("dodder://types/%s/objects", id),
			MimeType: "text/plain",
			Text:     result.Stdout,
		}},
	}, nil
}

func (p *typeResourceProvider) readTypeMarkl(
	ctx context.Context,
	id string,
) (*protocol.ResourceReadResult, error) {
	return p.readMarkl(
		ctx,
		"!" + id,
		fmt.Sprintf("dodder://types/%s/markl", id),
	)
}

func (p *typeResourceProvider) readObjectMarkl(
	ctx context.Context,
	objectId string,
) (*protocol.ResourceReadResult, error) {
	return p.readMarkl(
		ctx,
		objectId,
		fmt.Sprintf("dodder://objects/%s/markl", objectId),
	)
}

func (p *typeResourceProvider) readMarkl(
	ctx context.Context,
	queryId string,
	uri string,
) (*protocol.ResourceReadResult, error) {
	result, err := p.bridge.RunCommand(
		ctx,
		"show",
		[]string{"-format", "json", queryId},
		defaultMaxBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("show markl %s: %w", queryId, err)
	}

	for _, line := range strings.Split(result.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var full map[string]any
		if err := json.Unmarshal([]byte(line), &full); err != nil {
			continue
		}

		// Extract only the markl (merkle-tree) fields
		markl := map[string]any{
			"object-id":        full["object-id"],
			"object-digest":    full["object-digest"],
			"repo-pub_key":     full["repo-pub_key"],
			"repo-sig":         full["repo-sig"],
			"mother-object-sig": full["mother-object-sig"],
			"blob-id":          full["blob-id"],
		}

		output, err := json.MarshalIndent(markl, "", "  ")
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
	}

	return nil, fmt.Errorf("object not found: %s", queryId)
}

func (p *typeResourceProvider) readTypeBlobFormatted(
	ctx context.Context,
	id string,
	format string,
) (*protocol.ResourceReadResult, error) {
	result, err := p.bridge.RunCommand(
		ctx,
		"format-blob",
		[]string{format, "!" + id},
		defaultMaxBytes,
	)
	if err != nil {
		return p.readTypeBlob(ctx, id)
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      fmt.Sprintf("dodder://types/%s/blob/%s", id, format),
			MimeType: "text/plain",
			Text:     result.Stdout,
		}},
	}, nil
}

func (p *typeResourceProvider) readObjectBlob(
	ctx context.Context,
	objectId string,
	format string,
) (*protocol.ResourceReadResult, error) {
	result, err := p.bridge.RunCommand(
		ctx,
		"format-blob",
		[]string{format, objectId},
		defaultMaxBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("format-blob %s %s: %w", format, objectId, err)
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      fmt.Sprintf("dodder://objects/%s/blob/%s", objectId, format),
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
			Description: "Type metadata with links to blob, objects, and markl sub-resources.",
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

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder://types/{type_id}/blob/{format}",
			Name:        "Type Blob (Formatted)",
			Description: "Type blob content rendered with a specific formatter.",
			MimeType:    "text/plain",
		},
		nil,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder://types/{type_id}/objects",
			Name:        "Type Objects",
			Description: "All objects of this type in box format (one line per object). See server instructions for box format grammar. For blob content use dodder://objects/{id}/blob/{format}. For markl (merkle-tree) fields use dodder://objects/{id}/markl.",
			MimeType:    "text/plain",
		},
		nil,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder://types/{type_id}/markl",
			Name:        "Type Markl",
			Description: "Markl (merkle-tree) integrity fields for a type: object-digest, repo signature, repo public key, mother-object-sig, blob-id. Most queries do not need this — use only when verifying integrity or provenance.",
			MimeType:    "application/json",
		},
		nil,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder://objects/{object_id}/blob/{format}",
			Name:        "Object Blob (Formatted)",
			Description: "Object blob content rendered with a specific formatter (e.g. 'text').",
			MimeType:    "text/plain",
		},
		nil,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder://objects/{object_id}/markl",
			Name:        "Object Markl",
			Description: "Markl (merkle-tree) integrity fields for an object: object-digest, repo signature, repo public key, mother-object-sig, blob-id. Most queries do not need this — use only when verifying integrity or provenance.",
			MimeType:    "application/json",
		},
		nil,
	)
}
