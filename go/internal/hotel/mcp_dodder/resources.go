package mcp_dodder

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/amarbel-llc/purse-first/libs/go-mcp/protocol"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/server"
)

type typeResourceProvider struct {
	registry *server.ResourceRegistry
	index    *typeIndex
	tagIndex *tagIndex
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

		return p.readObject(ctx, rest)

	case strings.HasPrefix(uri, "dodder://types/"):
		rest := strings.TrimPrefix(uri, "dodder://types/")

		if strings.HasSuffix(rest, "/objects/facets") {
			id := strings.TrimSuffix(rest, "/objects/facets")
			return p.readTypeObjectFacets(ctx, id)
		}

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

	case strings.HasPrefix(uri, "dodder://tags/"):
		rest := strings.TrimPrefix(uri, "dodder://tags/")

		if strings.HasSuffix(rest, "/objects/facets") {
			id := strings.TrimSuffix(rest, "/objects/facets")
			return p.readTagObjectFacets(ctx, id)
		}

		if strings.HasSuffix(rest, "/objects") {
			id := strings.TrimSuffix(rest, "/objects")
			return p.readTagObjects(ctx, id)
		}

		if strings.HasSuffix(rest, "/markl") {
			id := strings.TrimSuffix(rest, "/markl")
			return p.readTagMarkl(ctx, id)
		}

		return p.readTag(ctx, rest)
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

func (p *typeResourceProvider) readTypeObjectFacets(
	ctx context.Context,
	id string,
) (*protocol.ResourceReadResult, error) {
	result, err := p.bridge.RunCommand(
		ctx,
		"show",
		[]string{"-format", "json", "!" + id},
		500_000,
	)
	if err != nil {
		return nil, fmt.Errorf("query type objects %s: %w", id, err)
	}

	type facetEntry struct {
		Value string `json:"value"`
		Count int    `json:"count"`
	}

	totalCount := 0
	tagCounts := make(map[string]int)
	prefixGroups := make(map[string]map[string]int)

	for _, line := range strings.Split(result.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var obj struct {
			Tags []string `json:"tags"`
		}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}

		totalCount++

		for _, tag := range obj.Tags {
			if strings.HasPrefix(tag, "-repo") {
				continue
			}
			tagCounts[tag]++

			if idx := strings.Index(tag, "-"); idx > 0 {
				prefix := tag[:idx]
				if prefixGroups[prefix] == nil {
					prefixGroups[prefix] = make(map[string]int)
				}
				prefixGroups[prefix][tag]++
			}
		}
	}

	// Build grouped facets
	type facetGroup struct {
		Prefix string       `json:"prefix"`
		Total  int          `json:"total"`
		Values []facetEntry `json:"values"`
	}

	var groups []facetGroup
	groupPrefixes := make([]string, 0, len(prefixGroups))
	for prefix := range prefixGroups {
		groupPrefixes = append(groupPrefixes, prefix)
	}
	sort.Strings(groupPrefixes)

	for _, prefix := range groupPrefixes {
		values := prefixGroups[prefix]
		entries := make([]facetEntry, 0, len(values))
		groupTotal := 0
		for value, count := range values {
			entries = append(entries, facetEntry{Value: value, Count: count})
			groupTotal += count
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Count > entries[j].Count
		})
		groups = append(groups, facetGroup{
			Prefix: prefix,
			Total:  groupTotal,
			Values: entries,
		})
	}

	// Collect ungrouped tags (no hyphen prefix)
	var ungrouped []facetEntry
	for tag, count := range tagCounts {
		if !strings.Contains(tag, "-") {
			ungrouped = append(ungrouped, facetEntry{Value: tag, Count: count})
		}
	}
	sort.Slice(ungrouped, func(i, j int) bool {
		return ungrouped[i].Count > ungrouped[j].Count
	})

	facets := struct {
		TotalObjects int          `json:"total_objects"`
		TagGroups    []facetGroup `json:"tag_groups"`
		Ungrouped    []facetEntry `json:"ungrouped_tags,omitempty"`
	}{
		TotalObjects: totalCount,
		TagGroups:    groups,
		Ungrouped:    ungrouped,
	}

	output, err := json.MarshalIndent(facets, "", "  ")
	if err != nil {
		return nil, err
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      fmt.Sprintf("dodder://types/%s/objects/facets", id),
			MimeType: "application/json",
			Text:     string(output),
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

func (p *typeResourceProvider) readObject(
	ctx context.Context,
	objectId string,
) (*protocol.ResourceReadResult, error) {
	result, err := p.bridge.RunCommand(
		ctx,
		"show",
		[]string{"-format", "json", objectId},
		defaultMaxBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("show object %s: %w", objectId, err)
	}

	for _, line := range strings.Split(result.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}

		// Build lightweight detail excluding markl fields
		detail := map[string]any{
			"object-id":   obj["object-id"],
			"date":        obj["date"],
			"description": obj["description"],
			"tags":        obj["tags"],
			"type":        obj["type"],
		}

		// Add traversal links
		typeId := ""
		if t, ok := obj["type"].(string); ok {
			typeId = strings.TrimPrefix(t, "!")
		}

		detail["blob-resource"] = fmt.Sprintf(
			"dodder://objects/%s/blob/text", objectId,
		)
		detail["markl-resource"] = fmt.Sprintf(
			"dodder://objects/%s/markl", objectId,
		)

		if typeId != "" {
			detail["type-resource"] = fmt.Sprintf(
				"dodder://types/%s", typeId,
			)
			detail["type-objects-resource"] = fmt.Sprintf(
				"dodder://types/%s/objects", typeId,
			)
		}

		// Add tag resource links for each tag
		if tags, ok := obj["tags"].([]any); ok && len(tags) > 0 {
			tagResources := make([]map[string]string, 0, len(tags))
			for _, t := range tags {
				if tag, ok := t.(string); ok {
					if strings.HasPrefix(tag, "-repo") {
						continue
					}
					stripped := strings.TrimPrefix(tag, "%")
					tagResources = append(tagResources, map[string]string{
						"tag":      tag,
						"resource": fmt.Sprintf("dodder://tags/%s", stripped),
					})
				}
			}
			detail["tag-resources"] = tagResources
		}

		output, err := json.MarshalIndent(detail, "", "  ")
		if err != nil {
			return nil, err
		}

		return &protocol.ResourceReadResult{
			Contents: []protocol.ResourceContent{{
				URI:      fmt.Sprintf("dodder://objects/%s", objectId),
				MimeType: "application/json",
				Text:     string(output),
			}},
		}, nil
	}

	return nil, fmt.Errorf("object not found: %s", objectId)
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

func (p *typeResourceProvider) readTag(
	ctx context.Context,
	id string,
) (*protocol.ResourceReadResult, error) {
	if err := p.tagIndex.ensureBuilt(); err != nil {
		return nil, fmt.Errorf("build tag index: %w", err)
	}

	results := p.tagIndex.query([]string{id})

	var found *tagSummary
	for i := range results {
		if results[i].ObjectId == id {
			found = &results[i]
			break
		}
	}

	if found == nil {
		return nil, fmt.Errorf("tag not found: %s", id)
	}

	detail := map[string]any{
		"object-id":        found.ObjectId,
		"date":             found.Date,
		"description":      found.Description,
		"tags":             found.Tags,
		"resource-uri":     found.ResourceURI,
		"objects-resource": fmt.Sprintf("dodder://tags/%s/objects", id),
		"markl-resource":   fmt.Sprintf("dodder://tags/%s/markl", id),
	}

	output, err := json.MarshalIndent(detail, "", "  ")
	if err != nil {
		return nil, err
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      fmt.Sprintf("dodder://tags/%s", id),
			MimeType: "application/json",
			Text:     string(output),
		}},
	}, nil
}

func (p *typeResourceProvider) readTagObjects(
	ctx context.Context,
	id string,
) (*protocol.ResourceReadResult, error) {
	result, err := p.bridge.RunCommand(
		ctx,
		"show",
		[]string{"-format", "box", id},
		500_000,
	)
	if err != nil {
		return nil, fmt.Errorf("query tag objects %s: %w", id, err)
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      fmt.Sprintf("dodder://tags/%s/objects", id),
			MimeType: "text/plain",
			Text:     result.Stdout,
		}},
	}, nil
}

func (p *typeResourceProvider) readTagObjectFacets(
	ctx context.Context,
	id string,
) (*protocol.ResourceReadResult, error) {
	result, err := p.bridge.RunCommand(
		ctx,
		"show",
		[]string{"-format", "json", id},
		500_000,
	)
	if err != nil {
		return nil, fmt.Errorf("query tag objects %s: %w", id, err)
	}

	type facetEntry struct {
		Value string `json:"value"`
		Count int    `json:"count"`
	}

	totalCount := 0
	tagCounts := make(map[string]int)
	prefixGroups := make(map[string]map[string]int)

	for _, line := range strings.Split(result.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var obj struct {
			Tags []string `json:"tags"`
		}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}

		totalCount++

		for _, tag := range obj.Tags {
			if strings.HasPrefix(tag, "-repo") {
				continue
			}
			tagCounts[tag]++

			if idx := strings.Index(tag, "-"); idx > 0 {
				prefix := tag[:idx]
				if prefixGroups[prefix] == nil {
					prefixGroups[prefix] = make(map[string]int)
				}
				prefixGroups[prefix][tag]++
			}
		}
	}

	type facetGroup struct {
		Prefix string       `json:"prefix"`
		Total  int          `json:"total"`
		Values []facetEntry `json:"values"`
	}

	var groups []facetGroup
	groupPrefixes := make([]string, 0, len(prefixGroups))
	for prefix := range prefixGroups {
		groupPrefixes = append(groupPrefixes, prefix)
	}
	sort.Strings(groupPrefixes)

	for _, prefix := range groupPrefixes {
		values := prefixGroups[prefix]
		entries := make([]facetEntry, 0, len(values))
		groupTotal := 0
		for value, count := range values {
			entries = append(entries, facetEntry{Value: value, Count: count})
			groupTotal += count
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Count > entries[j].Count
		})
		groups = append(groups, facetGroup{
			Prefix: prefix,
			Total:  groupTotal,
			Values: entries,
		})
	}

	var ungrouped []facetEntry
	for tag, count := range tagCounts {
		if !strings.Contains(tag, "-") {
			ungrouped = append(ungrouped, facetEntry{Value: tag, Count: count})
		}
	}
	sort.Slice(ungrouped, func(i, j int) bool {
		return ungrouped[i].Count > ungrouped[j].Count
	})

	facets := struct {
		TotalObjects int          `json:"total_objects"`
		TagGroups    []facetGroup `json:"tag_groups"`
		Ungrouped    []facetEntry `json:"ungrouped_tags,omitempty"`
	}{
		TotalObjects: totalCount,
		TagGroups:    groups,
		Ungrouped:    ungrouped,
	}

	output, err := json.MarshalIndent(facets, "", "  ")
	if err != nil {
		return nil, err
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      fmt.Sprintf("dodder://tags/%s/objects/facets", id),
			MimeType: "application/json",
			Text:     string(output),
		}},
	}, nil
}

func (p *typeResourceProvider) readTagMarkl(
	ctx context.Context,
	id string,
) (*protocol.ResourceReadResult, error) {
	return p.readMarkl(
		ctx,
		id,
		fmt.Sprintf("dodder://tags/%s/markl", id),
	)
}

func registerResources(
	registry *server.ResourceRegistry,
	index *typeIndex,
	tagIdx *tagIndex,
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
			URITemplate: "dodder://types/{type_id}/objects/facets",
			Name:        "Type Object Facets",
			Description: "Tag breakdown for all objects of this type, grouped by tag prefix (e.g. priority-, urgency-, area-). Returns total count and per-tag counts sorted by frequency. Start here for analytics before drilling into individual objects.",
			MimeType:    "application/json",
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
			URITemplate: "dodder://objects/{object_id}",
			Name:        "Object Detail",
			Description: "Object metadata (id, date, description, type, tags) with traversal links to blob, markl, type, and tag resources. Excludes heavy markl fields — use dodder://objects/{id}/markl for integrity data.",
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

	// Tag resources

	registry.RegisterResource(
		protocol.Resource{
			URI:         "dodder://tags_index",
			Name:        "Tag Word Index",
			Description: "Word list for tag discovery. Start here, then use tag_query tool or drill into dodder://tags/<id>.",
			MimeType:    "application/json",
		},
		func(ctx context.Context, uri string) (*protocol.ResourceReadResult, error) {
			if err := tagIdx.ensureBuilt(); err != nil {
				return nil, err
			}

			type wordEntry struct {
				Word  string `json:"word"`
				Count int    `json:"count"`
			}

			words := tagIdx.sortedWords()
			entries := make([]wordEntry, len(words))
			for i, w := range words {
				entries[i] = wordEntry{
					Word:  w,
					Count: len(tagIdx.words[w]),
				}
			}

			result := struct {
				TotalWords int         `json:"total_words"`
				TotalTags  int         `json:"total_tags"`
				Words      []wordEntry `json:"words"`
			}{
				TotalWords: len(words),
				TotalTags:  countUniqueTags(tagIdx),
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
			URI:         "dodder://tags",
			Name:        "All Tags",
			Description: "List of all tag objects with resource URIs. Use dodder://tags/<id> for full metadata.",
			MimeType:    "application/json",
		},
		func(ctx context.Context, uri string) (*protocol.ResourceReadResult, error) {
			if err := tagIdx.ensureBuilt(); err != nil {
				return nil, err
			}

			seen := make(map[string]bool)
			var tags []tagSummary

			for _, summaries := range tagIdx.words {
				for _, s := range summaries {
					if !seen[s.ObjectId] {
						seen[s.ObjectId] = true
						tags = append(tags, s)
					}
				}
			}

			output, err := json.MarshalIndent(tags, "", "  ")
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
			URITemplate: "dodder://tags/{tag_id}",
			Name:        "Tag Object",
			Description: "Tag metadata with links to objects and markl sub-resources.",
			MimeType:    "application/json",
		},
		nil,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder://tags/{tag_id}/objects",
			Name:        "Tag Objects",
			Description: "All objects with this tag in box format (one line per object). See server instructions for box format grammar.",
			MimeType:    "text/plain",
		},
		nil,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder://tags/{tag_id}/objects/facets",
			Name:        "Tag Object Facets",
			Description: "Tag breakdown for all objects with this tag, grouped by tag prefix. Returns total count and per-tag counts sorted by frequency.",
			MimeType:    "application/json",
		},
		nil,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder://tags/{tag_id}/markl",
			Name:        "Tag Markl",
			Description: "Markl (merkle-tree) integrity fields for a tag. Most queries do not need this.",
			MimeType:    "application/json",
		},
		nil,
	)
}
