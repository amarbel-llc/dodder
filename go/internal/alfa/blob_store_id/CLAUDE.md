# blob_store_id

Identifier type for blob stores with location-aware prefixes.

## Key Types

- `Id`: Blob store identifier with location and string ID
- `LocationType`: Enum for store location types (XDG user, etc.)

## Features

- Text marshaling/unmarshaling support
- Location-based prefixing for ID strings

## ID Format

A blob store ID has an optional location prefix followed by a name string.
Unprefixed IDs default to XDG user location.

| Prefix     | Location            | Example    | Filesystem root                         |
|------------|---------------------|------------|-----------------------------------------|
| *(none)*   | XDG user            | `default`  | `$XDG_DATA_HOME/madder/blob_stores/`    |
| `.`        | CWD                 | `.archive` | `$PWD/.madder/local/share/blob_stores/` |
| `/`        | XDG system          | `/system`  | system data dir                         |
| `_`        | Unknown             | `_custom`  | (custom path)                           |
| `~`        | *(backward compat)* | `~default` | same as unprefixed (parsed as XDG user) |

`Set()` checks if the first character is a known prefix (`.`, `/`, `_`, `~`).
If so, it splits prefix + remainder. Otherwise the entire value is the name with
XDG user location. `String()` omits the prefix for XDG user IDs. `~` is accepted
on parse for backward compatibility but is never emitted.

Two IDs with the same name but different locations (e.g. `default` vs `.default`)
refer to **different stores** at different filesystem locations. CWD stores
(`.` prefix) resolve relative to `$PWD`.
