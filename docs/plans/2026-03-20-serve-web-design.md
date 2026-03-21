# serve-web: Read-Only HTTP API for Dodder Stores

**Date:** 2026-03-20 **Status:** proposed

## Problem

Dodder has a rich, navigable resource tree exposed via MCP (`dodder://types`,
`dodder://objects/{id}`, etc.), but no way to serve that data over HTTP for
browser or frontend consumption. Sites like linenisgreat.com maintain a separate
PHP API layer that re-exports dodder data as JSON --- a build step shells out to
`der show`, pipes through `jq`, and writes static JSON files.

`serve-web` eliminates that intermediate layer by serving the MCP resource tree
directly over HTTP.

## Goals

- Serve dodder's existing MCP resources as an HTTP JSON API.
- Enable dodder.net as a self-hosted demo that serves its own repo.
- Keep the public-facing server separate from the repo-transfer server
  (`dodder serve`).

## Design

### Command

    dodder serve-web [network] [address]

Same syntax as `dodder serve`. Defaults to TCP on `:0`. Supports `tcp :8080`,
`unix /tmp/dodder-web.sock`.

Flag: `--cors-origin` (default `*`).

### Route → Resource Mapping

Each HTTP path maps 1:1 to a `dodder://` URI. The handler prepends `dodder://`
to the request path and calls `ReadResource`.

    GET /types_index                              → dodder://types_index
    GET /types                                    → dodder://types
    GET /types/{type_id}                          → dodder://types/{type_id}
    GET /types/{type_id}/blob                     → dodder://types/{type_id}/blob
    GET /types/{type_id}/blob/formats/{format_id} → dodder://types/{type_id}/blob/formats/{format_id}
    GET /types/{type_id}/objects                  → dodder://types/{type_id}/objects
    GET /types/{type_id}/objects/facets           → dodder://types/{type_id}/objects/facets
    GET /types/{type_id}/markl                    → dodder://types/{type_id}/markl
    GET /tags_index                               → dodder://tags_index
    GET /tags                                     → dodder://tags
    GET /tags/{tag_id}                            → dodder://tags/{tag_id}
    GET /tags/{tag_id}/objects                    → dodder://tags/{tag_id}/objects
    GET /tags/{tag_id}/objects/facets             → dodder://tags/{tag_id}/objects/facets
    GET /tags/{tag_id}/markl                      → dodder://tags/{tag_id}/markl
    GET /objects                                  → dodder://objects
    GET /objects/{object_id}                      → dodder://objects/{object_id}
    GET /objects/{object_id}/blob/formats         → dodder://objects/{object_id}/blob/formats
    GET /objects/{object_id}/blob/formats/{fmt}   → dodder://objects/{object_id}/blob/formats/{fmt}
    GET /objects/{object_id}/markl                → dodder://objects/{object_id}/markl
    GET /query/{terms...}                         → dodder://query/{terms...}

### Handler Logic

1.  Extract request path, prepend `dodder://`.
2.  Call `ResourceReader.ReadResource(ctx, uri)`.
3.  Write `Contents[0].Text` as response body.
4.  Set `Content-Type` from `Contents[0].MimeType`.
5.  Status 200.

Errors: resource not found → 404 `{"error": "not found: <uri>"}`. Internal error
→ 500 `{"error": "<message>"}`.

No response envelope --- the MCP resource JSON already includes counts and
metadata inline.

### CORS Middleware

Adds `Access-Control-Allow-Origin` from `--cors-origin` flag. `OPTIONS` requests
return 204 with CORS headers.

### New Resources

Two additions to `typeResourceProvider.ReadResource`:

**`dodder://objects`** --- All objects in box format. Implementation:
`bridge.RunCommand(ctx, "show", ["-format", "box", ":z", ":e", ":t"])`.
Registered as a static resource in `registerResources()`.

**`dodder://query/{terms}`** --- Path segments become AND-combined query terms.
Implementation: split path on `/`, pass segments to
`bridge.RunCommand(ctx, "show", append(["-format", "json"], terms...))`. Handled
via `strings.HasPrefix(uri, "dodder://query/")` in `ReadResource`.

### Exported Interface

Export from `mcp_dodder`:

``` go
type ResourceReader interface {
    ReadResource(ctx context.Context, uri string) (*protocol.ResourceReadResult, error)
}

func NewResourceReader(store *store.Store, bridge Bridge) ResourceReader
```

`serve-web` constructs a `ResourceReader` and uses only this method.

### Files

- `go/internal/victor/commands_dodder/serve_web.go` --- command registration
- `go/internal/tango/mcp_dodder/resources.go` --- export `ResourceReader`, add
  `dodder://objects` and `dodder://query/` handling
- `go/internal/tango/mcp_dodder/server_web.go` --- HTTP handler, router, CORS
  middleware

### No Authentication

Read-only, public-facing. No signature validation. The existing `dodder serve`
handles authenticated repo-transfer separately.

### No Pagination

The MCP resources don't paginate today. Future pagination belongs in the
resource provider itself, benefiting both MCP and web consumers.

## Testing

- BATS integration tests: start `dodder serve-web`, use `http` to hit endpoints,
  assert JSON structure.
- Add `serve-web` to `complete_subcmd` test in `complete.bats`.

## Rollback

Additive --- new command, no existing infrastructure replaced. If it doesn't
work, don't use it. For linenisgreat.com, the PHP API stays in place until
`serve-web` is proven.

## Out of Scope

- Replacing linenisgreat.com's PHP API
- dodder.net deployment/hosting
- Pagination
- TLS termination
- Static asset serving
