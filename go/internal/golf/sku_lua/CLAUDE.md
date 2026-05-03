# sku_lua

Lua table conversion for SKU objects (v1 and v2 formats).

## Purpose

Enables Lua scripting integration by converting SKU objects to/from Lua tables.

## Key Types

- `LuaTableV1`: Lua table structure with transacted data and tag tables
- `LuaTableV2`: Updated format with pool-based table management

## Features

- Bidirectional conversion between SKU objects and Lua tables
- Separate tables for explicit and implicit tags
- Pool-based Lua table reuse for v2 format
- Supports genre (Gattung), ID (Kennung), and type (Typ) fields
