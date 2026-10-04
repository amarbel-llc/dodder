# sku_lua

Lua table conversion for SKU objects (v1 and v2 formats).

## Purpose

Enables Lua scripting integration by converting SKU objects to/from Lua tables.

## Key Types

- `LuaTableV1`: Lua table structure with transacted data, tag tables, and the
  Fields projection (RFC-0006 Phase 1)
- `LuaTableV2`: English-keyed projection (Genre/ObjectId/Type/Tags instead of
  Gattung/Kennung/Typ/Etiketten), with the Fields projection and fields
  write-back at parity with V1 (#407)
- `ListTransformV1`: list handle backing `dodder.list()` for the
  inventory-list transform plugin (FDR-0024 / RFC-0008), with
  each()/remove()/add() and read-back via `FromLuaTableTransformV1`
- `ListTransformV2`: English-keyed list handle backing the opt-in
  `dodder.list_v2()` (RFC-0008 §3.6), lazily projected, read-back via
  `FromLuaTableTransformV2`

## Features

- Bidirectional conversion between SKU objects and Lua tables
- Separate tables for explicit and implicit tags
- Pool-based Lua table reuse
- Supports genre (Gattung), ID (Kennung), and type (Typ) fields
- `FromLuaTableTransformV1` additionally writes back Typ (safe only in the
  batch transform context, not in commit hooks -- issue #319)

## V1 vs V2 audit (Forgejo #385, 2026-08-09)

V2 does NOT supersede V1: V2 is a renamed-key projection that lacks the
Fields projection and fields write-back that RFC-0006 Phase 1 hooks and the
FDR-0024 transform both depend on. Every V1 surface has live users:

- `ToLuaTableV1`/`FromLuaTableV1`: commit hooks (`oscar/store/hooks.go`),
  `!toml-tag-v1`/`!lua-tag-v1` tag filters (`hotel/tag_blobs`),
  `local_working_copy/format_type.go`
- `LuaVMPoolV1`/`MakeLuaVMPoolV1`/`MakeLuaVMPoolV1WithSku`/`PushTopFuncV1`:
  hooks, `exec-lua` (`sierra/repo_actions/exec_lua.go`), format_type
- `ToLuaTableV1` via `ListTransformV1`: the `transform` command

The only V1 surface without users after the transform work,
`ToLuaArrayV1` (prototype-only), was deleted. Committed `!lua-tag-v1`
blobs remain decodable per the store-version decodable-forever rule.
Inverse finding: `FromLuaTableV2` currently has no callers (the `!lua-tag-v2`
filter path is read-only); kept as the V2 write-back counterpart.

## V2 parity (Forgejo #407)

The audit's gap is closed on the V2 side: `LuaTableV2` projects Fields,
`FromLuaTableV2` writes fields back (hook-safe: Type and Blob still withheld,
#319), and `ListTransformV2`/`FromLuaTableTransformV2` give the transform an
English-keyed binding. V1 is NOT yet deprecated and every V1 user above is
still on V1: migrating hooks, tag filters, `exec-lua`, and `format_type`, and
renaming keys in RFC-0006, wait on the take4 consolidation (#16), whose
transform script is written against the V1 keys. Until then do not rename or
remove V1 keys or change what `dodder.list()` returns. The V1 and V2
transform files duplicate each other on purpose (V1 files are left untouched
to avoid churn under take4); collapse them when V1 is retired.
