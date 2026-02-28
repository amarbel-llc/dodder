# file_extensions

Configurable file extension mappings for different object genres (zettel, tag, type, etc.).

## Key Types

- `Config`: File extensions for all supported genres
- `Overlay`: Optional overlay to customize extensions via TOML config
- `TOMLV0`, `TOMLV1`: Versioned TOML configuration formats

## Default Extensions

- Zettel: `.zettel`
- Tag: `.tag`
- Type: `.type`
- Repo: `.repo`
- Config: `.konfig`

## Features

- Genre-specific extension lookup via `GetFileExtensionForGenre`
- Overlay-based customization with `MakeConfig`
