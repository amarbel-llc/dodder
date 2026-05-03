# env_lua

Lua scripting environment for evaluating Lua scripts stored as blobs.

## Key Types

- `Env`: Lua environment interface with VM pool and SKU lookup
- `env`: Implementation with repo, object store, and format references

## Features

- Creates Lua VM pool builders with custom searchers
- Resolves object IDs from Lua require statements
- Loads and compiles Lua scripts from blob storage
- Integrates with gopher-lua for script execution
