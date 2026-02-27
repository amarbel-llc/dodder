# xdg_defaults

XDG Base Directory Specification defaults and template expansion.

## Key Types

- `DefaultEnvVar`: Environment variable with default and override templates

## Key Variables

- `Home`, `Cwd`: Basic directory variables
- `Data`, `Config`, `State`, `Cache`, `Runtime`: XDG directory defaults

## Features

- Template-based path expansion with $HOME, $XDG_OVERRIDE
- Override support for custom base directories
- Custom getenv function generation for utility-specific paths
