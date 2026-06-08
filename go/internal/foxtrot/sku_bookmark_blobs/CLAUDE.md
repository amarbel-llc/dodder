# sku_bookmark_blobs

Tommy-codegen leaf package for the bookmark TOML blob struct. Holds the
`//go:generate tommy generate` struct and its generated `*_tommy.go` so no
consumer of the generated `DecodeTomlBookmark` API shares this package (tommy
v0.4.3 codegen-isolation requirement).

## Key Types

- `TomlBookmark`: bookmark blob (a single `url` field)
- `TomlBookmarkDocument`: tommy document wrapper returned by `DecodeTomlBookmark`

## Consumer

`internal/golf/sku_json_fmt` re-exports `TomlBookmark`,
`TomlBookmarkDocument`, and `DecodeTomlBookmark` (the last as a `var`), and
owns `JsonWithUrl` (which embeds `TomlBookmark` through the alias). External
callers such as `sierra/store_browser` keep using `sku_json_fmt.*` unchanged.
