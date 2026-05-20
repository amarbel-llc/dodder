# Query Syntax Reference

Dodder uses a compact query syntax for selecting objects in commands like `show`,
`checkout`, `organize`, `edit`, `checkin`, `push`, `pull`, `clone`, and
`export`. Queries are passed as positional arguments.

## Basic Structure

A query argument has the general form:

```
[filter][sigil][genres]
```

All three parts are optional depending on context. Each command defines default
genres and sigils that apply when not explicitly specified.

## Object IDs

Specify an exact object by its two-part ID:

```bash
dodder show one/uno                # zettel with left-part "one", right-part "uno"
dodder show konfig                 # system configuration object
```

Object IDs can be used alongside genre-based queries:

```bash
dodder show one/uno,two/dos        # two specific objects (comma-separated)
```

## Genres

Genres identify the category of objects to select. Use single-character
abbreviations after a sigil.

| Abbreviation | Genre | Description |
|--------------|-------|-------------|
| `z` | Zettel | Notes and content objects |
| `t` | Tag | Tag objects |
| `e` | Type | Type definition objects |
| `r` | Repo | Remote repository objects |
| `b` | InventoryList | Inventory list objects |

The special identifier `konfig` selects the system configuration object
(genre: Config). It does not use a genre abbreviation.

### Multiple genres

Combine genres with commas:

```bash
dodder show :z,t                   # zettels and tags
dodder show :z,t,e                 # zettels, tags, and types
dodder show :z,t,e,r               # zettels, tags, types, and repos
```

### Default genres

Each command defines default genres applied when no genre is specified:

| Command | Default genres |
|---------|---------------|
| `show` | zettel |
| `checkout` | zettel |
| `organize` | zettel |
| `edit` | zettel, tag, type, repo |
| `checkin` | all |
| `status` | all |
| `clean` | all |
| `diff` | all |
| `push`, `pull` | inventory list |
| `clone` | inventory list |
| `export` | inventory list |
| `revert` | zettel, tag, type, repo |
| `fsck` | all |

## Sigils

Sigils control the scope of the query -- which versions and states of objects to
include.

| Character | Name | Description |
|-----------|------|-------------|
| `:` | Latest | Select only current (latest) versions |
| `?` | Hidden | Include dormant/hidden objects |
| `+` | History | Include historical (superseded) versions |
| `.` | External | Include checked-out/external objects |

Sigils appear between the filter and the genre:

```bash
dodder show :z                     # latest zettels
dodder show :?z                    # latest + dormant zettels
dodder show :+z                    # latest + historical zettels
```

### Default sigils

Each command defines default sigils:

| Command | Default sigil |
|---------|--------------|
| `show` | latest (`:`) |
| `checkout` | latest (`:`) |
| `organize` | latest (`:`) |
| `checkin` | external (`.`) |
| `status` | external (`.`) |
| `push`, `pull`, `clone` | history + hidden (`+?`) |
| `export` | history + hidden (`+?`) |
| `fsck` | latest + history + hidden (`:+?`) |

## Filtering by Tag

Prefix a genre selector with a tag name to filter objects by tag:

```bash
dodder show project:z              # zettels tagged "project"
dodder show work:z                 # zettels tagged "work"
dodder organize project:z          # organize only "project" zettels
dodder checkout work:z             # checkout only "work" zettels
```

The tag name goes before the sigil-genre pair. Only objects carrying that tag are
included.

## Filtering by Type

Prefix a genre selector with `!` followed by a type name:

```bash
dodder show !md:z                  # zettels of type "md"
dodder show !txt:z                 # zettels of type "txt"
dodder organize !md:z              # organize markdown zettels
dodder checkout !txt:z             # checkout text zettels
```

## Compound Queries

Combine multiple selectors by passing multiple positional arguments or using
commas:

```bash
dodder show one/uno two/dos        # two specific objects (separate arguments)
dodder show one/uno,tag-3,!md      # mixed: object ID, tag filter, type filter
dodder show project:z !md:z        # two separate filters (both apply)
```

Each positional argument is an independent query term. The results are the union
of all terms.

## Time Filtering

Use the `-before` and `-after` flags (on `show`) to filter by TAI timestamp.
Values are RFC3339 formatted.

```bash
dodder show -before 2024-06-01T00:00:00Z :z
dodder show -after 2024-01-01T00:00:00Z :z
dodder show -after 2024-01-01T00:00:00Z -before 2024-06-01T00:00:00Z :z
```

Time filtering is applied after the query matches, acting as a post-filter on
the result set.

## Workspace Default Query

When a workspace has a default query configured (via `init-workspace -query`),
that query is used when `show` is invoked without arguments. The workspace query
does not override explicit arguments.

```bash
dodder init-workspace -query "project:z"
dodder show                        # equivalent to: dodder show project:z
dodder show :t                     # explicit argument overrides workspace default
```

## Practical Examples

### Content management

| Goal | Command |
|------|---------|
| Show all markdown zettels | `dodder show !md:z` |
| Show zettels tagged "project" | `dodder show project:z` |
| Show all hidden/dormant zettels | `dodder show :?z` |
| Show all tags | `dodder show :t` |
| Show all types | `dodder show :e` |
| Show all remote repos | `dodder show :r` |
| Show system configuration | `dodder show konfig` |
| Show a specific zettel in detail | `dodder show -format text one/uno` |

### Bulk operations

| Goal | Command |
|------|---------|
| Checkout all zettels | `dodder checkout :z` |
| Checkout markdown zettels only | `dodder checkout !md:z` |
| Organize all zettels tagged "work" | `dodder organize work:z` |
| Organize all markdown zettels | `dodder organize !md:z` |
| Checkin everything in the workspace | `dodder checkin :z` |
| Clean unchanged checkouts | `dodder clean` |

### Remote operations

| Goal | Command |
|------|---------|
| Push everything to a remote | `dodder push remote-id` |
| Pull everything from a remote | `dodder pull remote-id` |
| Clone only markdown zettels | `dodder clone local-id remote:///path !md:z` |
| Export types and config | `dodder export :t,konfig` |

### History and dormant

| Goal | Command |
|------|---------|
| Show historical versions | `dodder show :+z` |
| Include dormant objects | `dodder show :?z` |
| Show all versions including dormant | `dodder show :+?z` |
| Fsck including all history | `dodder fsck :+?z` |

## Query Argument Grammar (Informal)

```
query       = object_id | filter_expr
object_id   = left "/" right
filter_expr = [tag_filter | type_filter] sigil genres
tag_filter  = tag_name
type_filter = "!" type_name
sigil       = ":" | "?" | "+" | "."
genres      = genre_char ("," genre_char)*
genre_char  = "z" | "t" | "e" | "r" | "b"
```

Special cases:

- `konfig` -- matches the Config genre directly
- Bare sigil + genres (`:z`) -- no filter, select by genre and scope
- Bare object ID (`one/uno`) -- exact match by identifier
- Comma-separated compound (`one/uno,tag-3,!md`) -- union of terms
