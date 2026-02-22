# blob_store_id

Identifier type for blob stores with location-aware prefixes.

## Key Types

- `Id`: Blob store identifier with location and string ID
- `LocationType`: Enum for store location types (XDG user, etc.)

## Features

- Text marshaling/unmarshaling support
- Location-based prefixing for ID strings

## ID Format

A blob store ID is a location prefix character followed by a name string. The
first character of the serialized form is always the location prefix:

| Prefix | Location            | Example      | Filesystem root                         |
|--------|---------------------|--------------|-----------------------------------------|
| `~`    | XDG user            | `~.default`  | `$XDG_DATA_HOME/madder/blob_stores/`    |
| `.`    | CWD (local overlay) | `..default`  | `./.madder/local/share/blob_stores/`    |
| `/`    | XDG system          | `/.default`  | system data dir                         |
| `_`    | Unknown             | `_.default`  | (custom path)                           |

`Set()` parses the first character as the prefix and the remainder as the name.
`String()` reconstructs `prefix + name`. Two IDs with the same name but
different prefixes (e.g. `~.default` vs `..default`) refer to **different
stores** at different filesystem locations. When `-override-xdg-with-cwd` is
active, both CWD and XDG user stores coexist in the same store map.
