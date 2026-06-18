package mcp_dodder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/repo_id"
	"code.linenisgreat.com/dodder/go/internal/golf/type_blobs"
	"code.linenisgreat.com/dodder/go/internal/oscar/store"
	"github.com/amarbel-llc/madder/go/pkgs/scoped_id"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/protocol"
	"github.com/amarbel-llc/purse-first/libs/go-mcp/server"
)

type typeResourceProvider struct {
	registry *server.ResourceRegistry
	bridge   Bridge

	// startupRepoId is the server's startup repo. The 3 store-backed
	// stragglers (readObjectBlobFormats / getBlobFormatIds, plus the edit
	// and reset-lock tools) hold this repo only; a repo-scoped read of a
	// different repo for them errors until the RepoManager follow-up
	// (FDR-0019, #278).
	startupRepoId scoped_id.Id

	// reposDir is the un-nested `<data>/repos/` directory whose
	// subdirectories name the repos in the active scope. Captured at
	// construction from the startup repo's nested data dir
	// (filepath.Dir(<data>/repos/<name>) == <data>/repos). Drives
	// readReposList without a command.Request (tango cannot import
	// uniform's listRepoNames).
	reposDir string

	store         *store.Store
	typeBlobCoder type_blobs.Coder

	// indexMu guards the lazily-populated per-repo index maps. The go-mcp
	// server dispatches each message on its own goroutine, so concurrent
	// reads of distinct repos must not race the map writes.
	indexMu     sync.Mutex
	typeIndexes map[string]*typeIndex
	tagIndexes  map[string]*tagIndex
}

// repoSeg is the `<repo>` path segment for a resolved repoId: its CLI
// spelling when explicit, else repo_id.DefaultName ("default") for the
// auto/zero id — so even an auto read emits a concrete repo-scoped link.
func repoSeg(repoId scoped_id.Id) string {
	if s := repoId.String(); s != "" {
		return s
	}

	return repo_id.DefaultName
}

// repoResourceURI builds the canonical repo-scoped resource URI
// dodder:///repos/<repo>/<suffix>.
func repoResourceURI(repoId scoped_id.Id, suffix string) string {
	return fmt.Sprintf("dodder:///repos/%s/%s", repoSeg(repoId), suffix)
}

func (p *typeResourceProvider) typeIndexFor(repoId scoped_id.Id) *typeIndex {
	key := repoSeg(repoId)

	p.indexMu.Lock()
	defer p.indexMu.Unlock()

	if p.typeIndexes == nil {
		p.typeIndexes = make(map[string]*typeIndex)
	}

	idx, ok := p.typeIndexes[key]
	if !ok {
		idx = makeTypeIndex(p.bridge, repoId)
		p.typeIndexes[key] = idx
	}

	return idx
}

func (p *typeResourceProvider) tagIndexFor(repoId scoped_id.Id) *tagIndex {
	key := repoSeg(repoId)

	p.indexMu.Lock()
	defer p.indexMu.Unlock()

	if p.tagIndexes == nil {
		p.tagIndexes = make(map[string]*tagIndex)
	}

	idx, ok := p.tagIndexes[key]
	if !ok {
		idx = makeTagIndex(p.bridge, repoId)
		p.tagIndexes[key] = idx
	}

	return idx
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
	// Canonical repo-scoped form: dodder:///repos[/<repo>[/<rest>]].
	// Both slashings are accepted defensively (triple-slash is the emitted
	// canonical form; the double-slash host-style form is tolerated).
	if rest, ok := trimReposPrefix(uri); ok {
		// rest is "" for the bare repos listing, "<repo>" for an
		// overview, or "<repo>/<kind>/..." for a per-kind read.
		if rest == "" {
			return p.readReposList(ctx)
		}

		repoSpelling, kindRest, hasKind := strings.Cut(rest, "/")

		var repoId scoped_id.Id
		if err := repoId.Set(repoSpelling); err != nil {
			return errorResourceResult(
				uri,
				fmt.Errorf("invalid repo_id %q: %w", repoSpelling, err),
			), nil
		}

		if err := repo_id.CheckSupported(repoId); err != nil {
			return errorResourceResult(uri, err), nil
		}

		if !hasKind || kindRest == "" {
			return p.readRepoOverview(ctx, repoId)
		}

		return p.dispatchKind(ctx, repoId, kindRest)
	}

	// Backward-compatible un-segmented form: dodder://<kind>/... resolves
	// to the auto/default repo (the server's startup repo). The zero-value
	// repoId yields "" from String(), which RunCommandWithRepoId resolves
	// to the server default.
	var autoRepoId scoped_id.Id

	switch {
	case uri == "dodder://objects":
		return p.registry.ReadResource(ctx, uri)

	case strings.HasPrefix(uri, "dodder://query/"):
		rest := strings.TrimPrefix(uri, "dodder://query/")
		terms := strings.Split(rest, "/")
		return p.readQuery(ctx, autoRepoId, terms)

	case strings.HasPrefix(uri, "dodder://objects/"):
		rest := strings.TrimPrefix(uri, "dodder://objects/")
		return p.dispatchObjects(ctx, autoRepoId, rest)

	case strings.HasPrefix(uri, "dodder://types/"):
		rest := strings.TrimPrefix(uri, "dodder://types/")
		return p.dispatchTypes(ctx, autoRepoId, rest)

	case strings.HasPrefix(uri, "dodder://tags/"):
		rest := strings.TrimPrefix(uri, "dodder://tags/")
		return p.dispatchTags(ctx, autoRepoId, rest)
	}

	return p.registry.ReadResource(ctx, uri)
}

// trimReposPrefix strips the repos collection prefix (triple- or
// double-slash) and reports the remaining path. For exactly the bare
// collection (dodder:///repos / dodder://repos) it returns ("", true).
func trimReposPrefix(uri string) (string, bool) {
	for _, prefix := range []string{"dodder:///repos", "dodder://repos"} {
		if uri == prefix {
			return "", true
		}
		if rest, ok := strings.CutPrefix(uri, prefix+"/"); ok {
			return rest, true
		}
	}

	return "", false
}

// errorResourceResult renders an error as a resource read result so a
// bad repo_id surfaces to the client as resource content rather than a
// transport-level failure.
func errorResourceResult(
	uri string,
	err error,
) *protocol.ResourceReadResult {
	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      uri,
			MimeType: "text/plain",
			Text:     fmt.Sprintf("error: %v", err),
		}},
	}
}

// dispatchKind routes the post-repo path of a repo-scoped URI to the
// same per-kind logic as the legacy un-segmented branches.
func (p *typeResourceProvider) dispatchKind(
	ctx context.Context,
	repoId scoped_id.Id,
	rest string,
) (*protocol.ResourceReadResult, error) {
	switch {
	case rest == "objects":
		return p.readObjectsListing(ctx, repoId)

	case rest == "types_index":
		return p.readTypesIndex(ctx, repoId)

	case rest == "tags_index":
		return p.readTagsIndex(ctx, repoId)

	case rest == "types":
		return p.readTypesListing(ctx, repoId)

	case rest == "tags":
		return p.readTagsListing(ctx, repoId)

	case strings.HasPrefix(rest, "query/"):
		terms := strings.Split(strings.TrimPrefix(rest, "query/"), "/")
		return p.readQuery(ctx, repoId, terms)

	case rest == "query":
		return p.readQuery(ctx, repoId, nil)

	case strings.HasPrefix(rest, "objects/"):
		return p.dispatchObjects(ctx, repoId, strings.TrimPrefix(rest, "objects/"))

	case strings.HasPrefix(rest, "types/"):
		return p.dispatchTypes(ctx, repoId, strings.TrimPrefix(rest, "types/"))

	case strings.HasPrefix(rest, "tags/"):
		return p.dispatchTags(ctx, repoId, strings.TrimPrefix(rest, "tags/"))
	}

	return nil, fmt.Errorf("resource not found: %s", rest)
}

func (p *typeResourceProvider) dispatchObjects(
	ctx context.Context,
	repoId scoped_id.Id,
	rest string,
) (*protocol.ResourceReadResult, error) {
	if before, after, ok := strings.Cut(rest, "/blob/formats/"); ok {
		objectId := before
		formatId := after
		return p.readObjectBlob(ctx, repoId, objectId, formatId)
	}

	if idx := strings.Index(rest, "/blob/formats"); idx >= 0 &&
		idx+len("/blob/formats") == len(rest) {
		objectId := rest[:idx]
		return p.readObjectBlobFormats(ctx, repoId, objectId)
	}

	if idx := strings.LastIndex(rest, "/markl"); idx >= 0 &&
		idx+len("/markl") == len(rest) {
		objectId := rest[:idx]
		return p.readObjectMarkl(ctx, repoId, objectId)
	}

	return p.readObject(ctx, repoId, rest)
}

func (p *typeResourceProvider) dispatchTypes(
	ctx context.Context,
	repoId scoped_id.Id,
	rest string,
) (*protocol.ResourceReadResult, error) {
	if before, ok := strings.CutSuffix(rest, "/objects/facets"); ok {
		return p.readTypeObjectFacets(ctx, repoId, before)
	}

	if before, ok := strings.CutSuffix(rest, "/objects"); ok {
		return p.readTypeObjects(ctx, repoId, before)
	}

	if before, ok := strings.CutSuffix(rest, "/markl"); ok {
		return p.readTypeMarkl(ctx, repoId, before)
	}

	if before, after, ok := strings.Cut(rest, "/blob/formats/"); ok {
		return p.readTypeBlobFormatted(ctx, repoId, before, after)
	}

	if before, ok := strings.CutSuffix(rest, "/blob"); ok {
		return p.readTypeBlob(ctx, repoId, before)
	}

	return p.readType(ctx, repoId, rest)
}

func (p *typeResourceProvider) dispatchTags(
	ctx context.Context,
	repoId scoped_id.Id,
	rest string,
) (*protocol.ResourceReadResult, error) {
	if before, ok := strings.CutSuffix(rest, "/objects/facets"); ok {
		return p.readTagObjectFacets(ctx, repoId, before)
	}

	if before, ok := strings.CutSuffix(rest, "/objects"); ok {
		return p.readTagObjects(ctx, repoId, before)
	}

	if before, ok := strings.CutSuffix(rest, "/markl"); ok {
		return p.readTagMarkl(ctx, repoId, before)
	}

	return p.readTag(ctx, repoId, rest)
}

func (p *typeResourceProvider) readType(
	ctx context.Context,
	repoId scoped_id.Id,
	id string,
) (*protocol.ResourceReadResult, error) {
	index := p.typeIndexFor(repoId)
	if err := index.ensureBuilt(); err != nil {
		return nil, fmt.Errorf("build type index: %w", err)
	}

	targetId := "!" + id
	results := index.query([]string{id})

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
		"blob-resource":    repoResourceURI(repoId, "types/"+id+"/blob"),
		"objects-resource": repoResourceURI(repoId, "types/"+id+"/objects"),
		"markl-resource":   repoResourceURI(repoId, "types/"+id+"/markl"),
	}

	output, err := json.MarshalIndent(detail, "", "  ")
	if err != nil {
		return nil, err
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      repoResourceURI(repoId, "types/"+id),
			MimeType: "application/json",
			Text:     string(output),
		}},
	}, nil
}

func (p *typeResourceProvider) readTypeBlob(
	ctx context.Context,
	repoId scoped_id.Id,
	id string,
) (*protocol.ResourceReadResult, error) {
	result, err := p.bridge.RunCommandWithRepoId(
		ctx,
		"show",
		[]string{"-format", "json-with-blob_string", "!" + id},
		defaultMaxBytes,
		repoId.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("show type blob %s: %w", id, err)
	}

	for line := range strings.SplitSeq(result.Stdout, "\n") {
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
				URI:      repoResourceURI(repoId, "types/"+id+"/blob"),
				MimeType: "text/plain",
				Text:     obj.BlobString,
			}},
		}, nil
	}

	return nil, fmt.Errorf("type %s has no blob content", id)
}

func (p *typeResourceProvider) readTypeObjects(
	ctx context.Context,
	repoId scoped_id.Id,
	id string,
) (*protocol.ResourceReadResult, error) {
	result, err := p.bridge.RunCommandWithRepoId(
		ctx,
		"show",
		[]string{"-format", "box", "!" + id},
		500_000,
		repoId.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("query type objects %s: %w", id, err)
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      repoResourceURI(repoId, "types/"+id+"/objects"),
			MimeType: "text/plain",
			Text:     result.Stdout,
		}},
	}, nil
}

func (p *typeResourceProvider) readTypeObjectFacets(
	ctx context.Context,
	repoId scoped_id.Id,
	id string,
) (*protocol.ResourceReadResult, error) {
	result, err := p.bridge.RunCommandWithRepoId(
		ctx,
		"show",
		[]string{"-format", "json", "!" + id},
		500_000,
		repoId.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("query type objects %s: %w", id, err)
	}

	output, err := computeFacets(result.Stdout)
	if err != nil {
		return nil, err
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      repoResourceURI(repoId, "types/"+id+"/objects/facets"),
			MimeType: "application/json",
			Text:     string(output),
		}},
	}, nil
}

func (p *typeResourceProvider) readTypeMarkl(
	ctx context.Context,
	repoId scoped_id.Id,
	id string,
) (*protocol.ResourceReadResult, error) {
	return p.readMarkl(
		ctx,
		repoId,
		"!"+id,
		repoResourceURI(repoId, "types/"+id+"/markl"),
	)
}

func (p *typeResourceProvider) readObjectMarkl(
	ctx context.Context,
	repoId scoped_id.Id,
	objectId string,
) (*protocol.ResourceReadResult, error) {
	return p.readMarkl(
		ctx,
		repoId,
		objectId,
		repoResourceURI(repoId, "objects/"+objectId+"/markl"),
	)
}

func (p *typeResourceProvider) readMarkl(
	ctx context.Context,
	repoId scoped_id.Id,
	queryId string,
	uri string,
) (*protocol.ResourceReadResult, error) {
	result, err := p.bridge.RunCommandWithRepoId(
		ctx,
		"show",
		[]string{"-format", "json", queryId},
		defaultMaxBytes,
		repoId.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("show markl %s: %w", queryId, err)
	}

	for line := range strings.SplitSeq(result.Stdout, "\n") {
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
			"object-id":         full["object-id"],
			"object-digest":     full["object-digest"],
			"repo-pub_key":      full["repo-pub_key"],
			"repo-sig":          full["repo-sig"],
			"mother-object-sig": full["mother-object-sig"],
			"blob-id":           full["blob-id"],
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
	repoId scoped_id.Id,
	id string,
	format string,
) (*protocol.ResourceReadResult, error) {
	result, err := p.bridge.RunCommandWithRepoId(
		ctx,
		"format-blob",
		[]string{"!" + id, format},
		defaultMaxBytes,
		repoId.String(),
	)
	if err != nil {
		return p.readTypeBlob(ctx, repoId, id)
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      repoResourceURI(repoId, "types/"+id+"/blob/formats/"+format),
			MimeType: "text/plain",
			Text:     result.Stdout,
		}},
	}, nil
}

func (p *typeResourceProvider) readObject(
	ctx context.Context,
	repoId scoped_id.Id,
	objectId string,
) (*protocol.ResourceReadResult, error) {
	result, err := p.bridge.RunCommandWithRepoId(
		ctx,
		"show",
		[]string{"-format", "json", objectId},
		defaultMaxBytes,
		repoId.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("show object %s: %w", objectId, err)
	}

	for line := range strings.SplitSeq(result.Stdout, "\n") {
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

		// Only link to blob formats if the object has blob content
		if blobId, _ := obj["blob-id"].(string); blobId != "" {
			detail["blob-formats-resource"] = repoResourceURI(
				repoId, "objects/"+objectId+"/blob/formats",
			)
		}

		detail["markl-resource"] = repoResourceURI(
			repoId, "objects/"+objectId+"/markl",
		)

		if typeId != "" {
			detail["type-resource"] = repoResourceURI(
				repoId, "types/"+typeId,
			)
			detail["type-objects-resource"] = repoResourceURI(
				repoId, "types/"+typeId+"/objects",
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
						"resource": repoResourceURI(repoId, "tags/"+stripped),
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
				URI:      repoResourceURI(repoId, "objects/"+objectId),
				MimeType: "application/json",
				Text:     string(output),
			}},
		}, nil
	}

	return nil, fmt.Errorf("object not found: %s", objectId)
}

func (p *typeResourceProvider) readObjectBlob(
	ctx context.Context,
	repoId scoped_id.Id,
	objectId string,
	format string,
) (*protocol.ResourceReadResult, error) {
	result, err := p.bridge.RunCommandWithRepoId(
		ctx,
		"format-blob",
		[]string{objectId, format},
		defaultMaxBytes,
		repoId.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("format-blob %s %s: %w", format, objectId, err)
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      repoResourceURI(repoId, "objects/"+objectId+"/blob/formats/"+format),
			MimeType: "text/plain",
			Text:     result.Stdout,
		}},
	}, nil
}

func (p *typeResourceProvider) readObjectBlobFormats(
	ctx context.Context,
	repoId scoped_id.Id,
	objectId string,
) (*protocol.ResourceReadResult, error) {
	formatIds, err := p.getBlobFormatIds(ctx, repoId, objectId)
	if err != nil {
		return nil, fmt.Errorf("blob formats for %s: %w", objectId, err)
	}

	type formatEntry struct {
		FormatId    string `json:"format_id"`
		ResourceURI string `json:"resource_uri"`
	}

	entries := make([]formatEntry, len(formatIds))
	for i, id := range formatIds {
		entries[i] = formatEntry{
			FormatId:    id,
			ResourceURI: repoResourceURI(repoId, "objects/"+objectId+"/blob/formats/"+id),
		}
	}

	output, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return nil, err
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      repoResourceURI(repoId, "objects/"+objectId+"/blob/formats"),
			MimeType: "application/json",
			Text:     string(output),
		}},
	}, nil
}

// getBlobFormatIds resolves the available blob formatter IDs for an object
// by reading its type object directly from the store, bypassing workspace
// query filters that would otherwise exclude type objects.
//
// This is one of the 3 stragglers (FDR-0019 RepoManager follow-up, #278):
// it touches the startup repo's p.store/p.typeBlobCoder directly, so it
// cannot be routed per-repo without a stateful repo-open-by-id
// (RepoManager). A read of a non-default repo therefore errors with a
// clear deferral message rather than silently returning the startup
// repo's formats.
func (p *typeResourceProvider) getBlobFormatIds(
	ctx context.Context,
	repoId scoped_id.Id,
	objectId string,
) ([]string, error) {
	if repoId.String() != p.startupRepoId.String() {
		return nil, fmt.Errorf(
			"per-repo not yet supported for blob-format listing on repo %q (RepoManager follow-up, see #278)",
			repoSeg(repoId),
		)
	}

	oid, oidRepool, err := ids.MakeObjectId(objectId)
	if err != nil {
		return nil, fmt.Errorf("parse object id %s: %w", objectId, err)
	}
	defer oidRepool()

	object, err := p.store.ReadTransactedFromObjectId(oid)
	if err != nil {
		return nil, fmt.Errorf("read object %s: %w", objectId, err)
	}

	typeObject, err := p.store.ReadObjectTypeAndLockIfNecessary(object)
	if err != nil {
		if errors.IsErrNotFound(err) {
			return nil, fmt.Errorf("type %s has no blob", object.GetType())
		}
		return nil, fmt.Errorf("read type for %s: %w", objectId, err)
	}

	blobDigest := typeObject.GetMetadata().GetBlobDigest()
	if blobDigest.IsNull() {
		return nil, fmt.Errorf("type %s has no blob", object.GetType())
	}

	typeBlob, repool, _, err := p.typeBlobCoder.ParseTypedBlob(typeObject.GetType(), blobDigest)
	if err != nil {
		return nil, fmt.Errorf("parse type blob for %s: %w", object.GetType(), err)
	}
	defer repool()

	formatters := typeBlob.GetFormatters()
	resultIds := make([]string, 0, len(formatters))
	for id := range formatters {
		resultIds = append(resultIds, id)
	}
	sort.Strings(resultIds)

	return resultIds, nil
}

func (p *typeResourceProvider) readTag(
	ctx context.Context,
	repoId scoped_id.Id,
	id string,
) (*protocol.ResourceReadResult, error) {
	tagIdx := p.tagIndexFor(repoId)
	if err := tagIdx.ensureBuilt(); err != nil {
		return nil, fmt.Errorf("build tag index: %w", err)
	}

	results := tagIdx.query([]string{id})

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
		"objects-resource": repoResourceURI(repoId, "tags/"+id+"/objects"),
		"markl-resource":   repoResourceURI(repoId, "tags/"+id+"/markl"),
	}

	output, err := json.MarshalIndent(detail, "", "  ")
	if err != nil {
		return nil, err
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      repoResourceURI(repoId, "tags/"+id),
			MimeType: "application/json",
			Text:     string(output),
		}},
	}, nil
}

func (p *typeResourceProvider) readTagObjects(
	ctx context.Context,
	repoId scoped_id.Id,
	id string,
) (*protocol.ResourceReadResult, error) {
	result, err := p.bridge.RunCommandWithRepoId(
		ctx,
		"show",
		[]string{"-format", "box", id},
		500_000,
		repoId.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("query tag objects %s: %w", id, err)
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      repoResourceURI(repoId, "tags/"+id+"/objects"),
			MimeType: "text/plain",
			Text:     result.Stdout,
		}},
	}, nil
}

func (p *typeResourceProvider) readTagObjectFacets(
	ctx context.Context,
	repoId scoped_id.Id,
	id string,
) (*protocol.ResourceReadResult, error) {
	result, err := p.bridge.RunCommandWithRepoId(
		ctx,
		"show",
		[]string{"-format", "json", id},
		500_000,
		repoId.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("query tag objects %s: %w", id, err)
	}

	output, err := computeFacets(result.Stdout)
	if err != nil {
		return nil, err
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      repoResourceURI(repoId, "tags/"+id+"/objects/facets"),
			MimeType: "application/json",
			Text:     string(output),
		}},
	}, nil
}

// computeFacets renders the tag-prefix facet breakdown for a stream of
// json objects (one per line). Shared by the type and tag facet handlers.
func computeFacets(stdout string) ([]byte, error) {
	type facetEntry struct {
		Value string `json:"value"`
		Count int    `json:"count"`
	}

	totalCount := 0
	tagCounts := make(map[string]int)
	prefixGroups := make(map[string]map[string]int)

	for line := range strings.SplitSeq(stdout, "\n") {
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

	return json.MarshalIndent(facets, "", "  ")
}

func (p *typeResourceProvider) readTagMarkl(
	ctx context.Context,
	repoId scoped_id.Id,
	id string,
) (*protocol.ResourceReadResult, error) {
	return p.readMarkl(
		ctx,
		repoId,
		id,
		repoResourceURI(repoId, "tags/"+id+"/markl"),
	)
}

func (p *typeResourceProvider) readQuery(
	ctx context.Context,
	repoId scoped_id.Id,
	terms []string,
) (*protocol.ResourceReadResult, error) {
	args := append([]string{"-format", "json"}, terms...)
	result, err := p.bridge.RunCommandWithRepoId(
		ctx,
		"show",
		args,
		500_000,
		repoId.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("query %v: %w", terms, err)
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      repoResourceURI(repoId, "query/"+strings.Join(terms, "/")),
			MimeType: "application/json",
			Text:     result.Stdout,
		}},
	}, nil
}

// readObjectsListing lists every object in the addressed repo (box
// format). The repo-scoped analog of the un-segmented dodder://objects
// registered resource.
func (p *typeResourceProvider) readObjectsListing(
	ctx context.Context,
	repoId scoped_id.Id,
) (*protocol.ResourceReadResult, error) {
	result, err := p.bridge.RunCommandWithRepoId(
		ctx,
		"show",
		[]string{"-format", "box", ":z", ":e", ":t"},
		500_000,
		repoId.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("list all objects: %w", err)
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      repoResourceURI(repoId, "objects"),
			MimeType: "text/plain",
			Text:     result.Stdout,
		}},
	}, nil
}

func (p *typeResourceProvider) readTypesIndex(
	ctx context.Context,
	repoId scoped_id.Id,
) (*protocol.ResourceReadResult, error) {
	index := p.typeIndexFor(repoId)
	if err := index.ensureBuilt(); err != nil {
		return nil, err
	}

	output, err := marshalTypeWordIndex(index)
	if err != nil {
		return nil, err
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      repoResourceURI(repoId, "types_index"),
			MimeType: "application/json",
			Text:     string(output),
		}},
	}, nil
}

func (p *typeResourceProvider) readTagsIndex(
	ctx context.Context,
	repoId scoped_id.Id,
) (*protocol.ResourceReadResult, error) {
	tagIdx := p.tagIndexFor(repoId)
	if err := tagIdx.ensureBuilt(); err != nil {
		return nil, err
	}

	output, err := marshalTagWordIndex(tagIdx)
	if err != nil {
		return nil, err
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      repoResourceURI(repoId, "tags_index"),
			MimeType: "application/json",
			Text:     string(output),
		}},
	}, nil
}

func (p *typeResourceProvider) readTypesListing(
	ctx context.Context,
	repoId scoped_id.Id,
) (*protocol.ResourceReadResult, error) {
	index := p.typeIndexFor(repoId)
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
			URI:      repoResourceURI(repoId, "types"),
			MimeType: "application/json",
			Text:     string(output),
		}},
	}, nil
}

func (p *typeResourceProvider) readTagsListing(
	ctx context.Context,
	repoId scoped_id.Id,
) (*protocol.ResourceReadResult, error) {
	tagIdx := p.tagIndexFor(repoId)
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
			URI:      repoResourceURI(repoId, "tags"),
			MimeType: "application/json",
			Text:     string(output),
		}},
	}, nil
}

// readReposList enumerates the repos in the active scope by scanning the
// captured un-nested <data>/repos/ directory, returning one entry per
// subdirectory with its repo-scoped overview URI. mcp_dodder is tango
// tier and cannot import uniform's listRepoNames, so this reimplements
// the directory scan from the captured base path.
func (p *typeResourceProvider) readReposList(
	ctx context.Context,
) (*protocol.ResourceReadResult, error) {
	type repoEntry struct {
		Name        string `json:"name"`
		ResourceURI string `json:"resource_uri"`
	}

	var names []string

	if p.reposDir != "" {
		entries, err := os.ReadDir(p.reposDir)
		if err != nil && !errors.IsNotExist(err) {
			return nil, fmt.Errorf("read repos dir %s: %w", p.reposDir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				names = append(names, entry.Name())
			}
		}
	}

	sort.Strings(names)

	repos := make([]repoEntry, len(names))
	for i, name := range names {
		repos[i] = repoEntry{
			Name:        name,
			ResourceURI: fmt.Sprintf("dodder:///repos/%s", name),
		}
	}

	doc := struct {
		TotalRepos int         `json:"total_repos"`
		Repos      []repoEntry `json:"repos"`
	}{
		TotalRepos: len(repos),
		Repos:      repos,
	}

	output, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      "dodder:///repos",
			MimeType: "application/json",
			Text:     string(output),
		}},
	}, nil
}

// readRepoOverview returns the link hub for a single repo: its
// objects/types/tags listings and word indexes.
func (p *typeResourceProvider) readRepoOverview(
	ctx context.Context,
	repoId scoped_id.Id,
) (*protocol.ResourceReadResult, error) {
	doc := map[string]any{
		"repo":             repoSeg(repoId),
		"objects-resource": repoResourceURI(repoId, "objects"),
		"types-resource":   repoResourceURI(repoId, "types"),
		"tags-resource":    repoResourceURI(repoId, "tags"),
		"types-index":      repoResourceURI(repoId, "types_index"),
		"tags-index":       repoResourceURI(repoId, "tags_index"),
	}

	output, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}

	return &protocol.ResourceReadResult{
		Contents: []protocol.ResourceContent{{
			URI:      fmt.Sprintf("dodder:///repos/%s", repoSeg(repoId)),
			MimeType: "application/json",
			Text:     string(output),
		}},
	}, nil
}

func marshalTypeWordIndex(index *typeIndex) ([]byte, error) {
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

	return json.MarshalIndent(result, "", "  ")
}

func marshalTagWordIndex(tagIdx *tagIndex) ([]byte, error) {
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

	return json.MarshalIndent(result, "", "  ")
}

// registerResources registers the static resources and parameterized
// templates the provider serves. The static-resource closures defer to
// the provider's ReadResource dispatch so the type/tag listings and word
// indexes resolve their per-repo index from the URI rather than a
// captured singleton; the legacy un-segmented forms keep resolving to the
// auto/default repo. The default repo's listings/indexes are also
// registered as concrete resources under their repo-scoped URIs.
func registerResources(
	registry *server.ResourceRegistry,
	provider *typeResourceProvider,
) {
	defaultSeg := repo_id.DefaultName

	dispatch := func(
		ctx context.Context,
		uri string,
	) (*protocol.ResourceReadResult, error) {
		return provider.ReadResource(ctx, uri)
	}

	// Repos collection + overview.

	registry.RegisterResource(
		protocol.Resource{
			URI:         "dodder:///repos",
			Name:        "Repos",
			Description: "List of all dodder repos in the active scope. Drill into dodder:///repos/<repo> for a repo overview.",
			MimeType:    "application/json",
		},
		dispatch,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder:///repos/{repo_id}",
			Name:        "Repo Overview",
			Description: "Overview of one repo with links to its objects, types, tags, and word indexes.",
			MimeType:    "application/json",
		},
		nil,
	)

	// Type resources (repo-scoped templates + default-repo concretes).

	registry.RegisterResource(
		protocol.Resource{
			URI:         fmt.Sprintf("dodder:///repos/%s/types_index", defaultSeg),
			Name:        "Type Word Index",
			Description: "Word list for type discovery. Start here, then use type_query tool or drill into dodder:///repos/<repo>/types/<id>.",
			MimeType:    "application/json",
		},
		dispatch,
	)

	registry.RegisterResource(
		protocol.Resource{
			URI:         fmt.Sprintf("dodder:///repos/%s/types", defaultSeg),
			Name:        "All Types",
			Description: "List of all type objects with resource URIs. Use dodder:///repos/<repo>/types/<id> for full metadata.",
			MimeType:    "application/json",
		},
		dispatch,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder:///repos/{repo_id}/types/{type_id}",
			Name:        "Type Object",
			Description: "Type metadata with links to blob, objects, and markl sub-resources.",
			MimeType:    "application/json",
		},
		nil,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder:///repos/{repo_id}/types/{type_id}/blob",
			Name:        "Type Blob",
			Description: "Type blob content (TOML configuration).",
			MimeType:    "text/plain",
		},
		nil,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder:///repos/{repo_id}/types/{type_id}/blob/formats/{format_id}",
			Name:        "Type Blob (Formatted)",
			Description: "Type blob content rendered with a specific formatter.",
			MimeType:    "text/plain",
		},
		nil,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder:///repos/{repo_id}/types/{type_id}/objects/facets",
			Name:        "Type Object Facets",
			Description: "Tag breakdown for all objects of this type, grouped by tag prefix (e.g. priority-, urgency-, area-). Returns total count and per-tag counts sorted by frequency. Start here for analytics before drilling into individual objects.",
			MimeType:    "application/json",
		},
		nil,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder:///repos/{repo_id}/types/{type_id}/objects",
			Name:        "Type Objects",
			Description: "All objects of this type in box format (one line per object). See server instructions for box format grammar. For blob content use dodder:///repos/<repo>/objects/{id}/blob/{format}. For markl (merkle-tree) fields use dodder:///repos/<repo>/objects/{id}/markl.",
			MimeType:    "text/plain",
		},
		nil,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder:///repos/{repo_id}/types/{type_id}/markl",
			Name:        "Type Markl",
			Description: "Markl (merkle-tree) integrity fields for a type: object-digest, repo signature, repo public key, mother-object-sig, blob-id. Most queries do not need this — use only when verifying integrity or provenance.",
			MimeType:    "application/json",
		},
		nil,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder:///repos/{repo_id}/objects/{object_id}",
			Name:        "Object Detail",
			Description: "Object metadata (id, date, description, type, tags) with traversal links to blob, markl, type, and tag resources. Excludes heavy markl fields — use dodder:///repos/<repo>/objects/{id}/markl for integrity data.",
			MimeType:    "application/json",
		},
		nil,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder:///repos/{repo_id}/objects/{object_id}/blob/formats",
			Name:        "Object Blob Formats",
			Description: "Lists available blob formatter IDs for this object's type, with resource URIs for each format.",
			MimeType:    "application/json",
		},
		nil,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder:///repos/{repo_id}/objects/{object_id}/blob/formats/{format_id}",
			Name:        "Object Blob (Formatted)",
			Description: "Object blob content rendered with a specific formatter.",
			MimeType:    "text/plain",
		},
		nil,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder:///repos/{repo_id}/objects/{object_id}/markl",
			Name:        "Object Markl",
			Description: "Markl (merkle-tree) integrity fields for an object: object-digest, repo signature, repo public key, mother-object-sig, blob-id. Most queries do not need this — use only when verifying integrity or provenance.",
			MimeType:    "application/json",
		},
		nil,
	)

	// Tag resources.

	registry.RegisterResource(
		protocol.Resource{
			URI:         fmt.Sprintf("dodder:///repos/%s/tags_index", defaultSeg),
			Name:        "Tag Word Index",
			Description: "Word list for tag discovery. Start here, then use tag_query tool or drill into dodder:///repos/<repo>/tags/<id>.",
			MimeType:    "application/json",
		},
		dispatch,
	)

	registry.RegisterResource(
		protocol.Resource{
			URI:         fmt.Sprintf("dodder:///repos/%s/tags", defaultSeg),
			Name:        "All Tags",
			Description: "List of all tag objects with resource URIs. Use dodder:///repos/<repo>/tags/<id> for full metadata.",
			MimeType:    "application/json",
		},
		dispatch,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder:///repos/{repo_id}/tags/{tag_id}",
			Name:        "Tag Object",
			Description: "Tag metadata with links to objects and markl sub-resources.",
			MimeType:    "application/json",
		},
		nil,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder:///repos/{repo_id}/tags/{tag_id}/objects",
			Name:        "Tag Objects",
			Description: "All objects with this tag in box format (one line per object). See server instructions for box format grammar.",
			MimeType:    "text/plain",
		},
		nil,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder:///repos/{repo_id}/tags/{tag_id}/objects/facets",
			Name:        "Tag Object Facets",
			Description: "Tag breakdown for all objects with this tag, grouped by tag prefix. Returns total count and per-tag counts sorted by frequency.",
			MimeType:    "application/json",
		},
		nil,
	)

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder:///repos/{repo_id}/tags/{tag_id}/markl",
			Name:        "Tag Markl",
			Description: "Markl (merkle-tree) integrity fields for a tag. Most queries do not need this.",
			MimeType:    "application/json",
		},
		nil,
	)

	// Object listing — un-segmented backward-compatible form (auto repo).

	registry.RegisterResource(
		protocol.Resource{
			URI:         "dodder://objects",
			Name:        "All Objects",
			Description: "List of all objects in box format (auto/default repo). See server instructions for box format grammar.",
			MimeType:    "text/plain",
		},
		func(ctx context.Context, uri string) (*protocol.ResourceReadResult, error) {
			var autoRepoId scoped_id.Id
			return provider.readObjectsListing(ctx, autoRepoId)
		},
	)

	registry.RegisterResource(
		protocol.Resource{
			URI:         fmt.Sprintf("dodder:///repos/%s/objects", defaultSeg),
			Name:        "All Objects",
			Description: "List of all objects in the default repo in box format. See server instructions for box format grammar.",
			MimeType:    "text/plain",
		},
		dispatch,
	)

	// Query.

	registry.RegisterTemplate(
		protocol.ResourceTemplate{
			URITemplate: "dodder:///repos/{repo_id}/query/{terms}",
			Name:        "Query",
			Description: "Execute a dodder query against a repo. Path segments after query/ are AND-combined query terms. Returns results in JSON format.",
			MimeType:    "application/json",
		},
		nil,
	)
}
