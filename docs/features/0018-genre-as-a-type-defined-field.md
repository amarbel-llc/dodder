---
status: exploring
date: 2026-06-02
promotion-criteria: |
  Promote to `proposed` once the three remaining sub-decisions on the
  default-query model are settled: (1) whether the single layered
  default-query model (AND of repo + workspace + user, with
  more-specific layers overriding on single-valued fields like genre and
  accumulating on multi-valued fields like tags) fully replaces today's
  two mechanisms, or a separate always-AND "scope" slot is also kept;
  (2) the precise override/suppression semantics across a boolean
  expression and the set of single-valued fields; (3) the config schema
  — a shared config-overlay (Defaults + default-query) embedded by both
  the repo mutable config and the workspace config, plus the version
  bumps and migration of the existing workspace `query` field into it.
  (Field lifting from a type's type-of remains explicitly deferred and
  is NOT a blocker.) Promote to `experimental` once a projected `_genre`
  field exists on committed objects, `der show _genre=zettel` returns the
  same set as today's `der show :z`, the per-genre index partition is
  flattened, and id-syntax genre derivation (`ValidateSeqAndGetGenre`)
  is removed in favor of the field.
---

# Genre as a Type-Defined Field

> **Status (2026-06-02):** Exploring. This FDR records design intent for
> dissolving the standalone *genre* concept into the *type* and *field*
> systems that already exist. No code has changed. The mechanism is
> settled in outline — genre is a field a type declares, decoupled from
> id syntax, projected into the index, migrated via `import`. The
> default-query / workspace-scope interaction (the one design point that
> looked unresolved) is now worked out: a single layered default-query
> model where the apparent "fallback vs AND" tension turns out to be
> field cardinality (single- vs multi-valued), not a real conflict. What
> remains are schema/semantics sub-decisions. Field lifting is deferred.
> Builds on the type-of chain (whose fixed point is FDR-0010's null type
> `!`) and the fields infrastructure shipped on the mild-elm branch.
>
> **Field spelling (2026-07-18):** the projected field is spelled
> `_genre`, not bare `genre`, aligning with cutting-garden RFC 0014's
> trellis grammar, where framework-reserved virtual fields are
> `_`-prefixed (`_genre`, `_body`, `_description`) and a leading
> underscore is illegal for type-declared field names — this keeps bare
> `genre` free for real per-type data (jira/caldav-style fields) and
> keeps the trellis/doddish superset relationship exact. See dodder#373.

## Problem Statement

Genre is a first-class, standalone concept in dodder: a `genres.Genre`
enum (Zettel, Tag, Type, Repo, Blob, Config, InventoryList), a parallel
`ids.Genre` bitfield, and a sigil-suffix query grammar
(`tag:z,e,t,k`). It is **derived from object-id syntax** at parse time
(`ValidateSeqAndGetGenre`, `go/internal/bravo/ids/main.go:258-361`) and
**never persisted** — the id shape *is* the genre. This makes genre a
third axis of classification beside the type system instead of inside
it, and it forces the query language into a shape no other predicate
has.

Two concrete costs:

1. **Genre is divorced from types, yet types already imply it.** A
   builtin type already asserts the genre of its instances —
   `BuiltinType` embeds a `genres.Genre`
   (`go/internal/bravo/ids/types_builtin.go:59-63`), and registration
   encodes the assertion: `…(TypeTomlTagV1, genres.Tag, …)` →
   "an object typed `!toml-tag-v1` is a Tag"; `…(TypeTomlTypeV2,
   genres.Type, …)` → "an object typed `!toml-type-v2` is a Type"
   (`:116-120`). But this is registered only for builtins, used only in
   the genre→default-type direction (`DefaultOrPanic`), and *ignored* at
   query time in favor of re-deriving genre from id syntax. User types
   like `!md` cannot make this assertion at all.

2. **The query grammar groups by genre instead of composing.** A query
   term is `<predicate><sigil><genre-suffix>`. The genre suffix causes
   the *entire predicate tree to be cloned per genre* into
   `optimizedQueries[g]` (`go/internal/juliett/queries/main.go:124-152`).
   Genre is a grouping axis, not a predicate: it cannot participate in
   the boolean algebra the rest of the query language uses. "Objects
   that are (zettels or tags) and have tag X" has no natural form. This
   is unorthodox and is the root of the "each genre gets its own copy of
   the whole query" problem.

## The insight: genre is already a field, queried like one

Dodder already has a *fields* system and a *field-match query syntax*,
and field predicates already compose natively:

- **Fields are declared by types.** A type blob carries `[[fields]]`
  (`FieldDefinition`: `name`, `kind`, `values`, `default`;
  `go/internal/alfa/type_blobs/field_definition.go:6-11`), kinds
  `string | enum | bool | u32 | s32 | list<string>`
  (`go/internal/0/fields/main.go:19-26`).
- **Field values live on the object, in the index.** Projected into
  `Metadata.Index.Fields` as `Field{Type, Key, Value, TypeBlobDigest}`
  (`go/internal/0/fields/main.go:54-58`) at commit time and indexed by
  default.
- **Field-match is an ordinary, composable predicate.** `key=value` and
  `^key=value` parse into an `expField` added to the boolean stack
  (`go/internal/juliett/queries/build_state.go:262-279`,
  `exp_field.go`). Unlike the genre suffix, it is just another node in
  the expression tree — it composes with `&`, `|`, and negation for
  free, no per-value duplication.
- **`type` is already a (display-only) virtual field**
  (`fields.TypeType`, `go/internal/echo/object_metadata_fmt/fields.go:60-69`).
  `_genre` is its natural sibling.

So the redesign is: **make `_genre` a field** a type declares, and let
queries discriminate genre with the field-match syntax that already
exists. The standalone genre enum/bitfield/sigil-suffix grammar
collapses into type + field.

## Design direction

### Types are recursive blob parsers; genre is a field they declare

The mental model is not class inheritance. **A type is a
parser/reader/writer/serializer of a blob shape, applied recursively.**
A type reads its instances' blobs into the expressive type system —
projecting fields, validating, formatting. A type is itself an object
with a blob, so *its* type reads *its* blob: `!md` reads markdown notes;
the `!md` type object's own blob is TOML, read by `!toml-type-v2`. The
chain (note → `!md` → `!toml-type-v2` → …) is the "what parses this
blob" recursion, terminating at the null type `!` (FDR-0010) — raw bytes
that need no parser.

`_genre` is one of the fields a type declares — a value the type asserts
for **all** its instances (every `!md` note is `_genre=zettel`,
regardless of content), as distinct from per-instance content fields
like `status` that a `fields-reader` extracts from each blob.

### Flat composition; field lifting deferred

Field composition across types is **flat**: each type declares its own
fields directly, including `_genre`. `!task` and `!chore` each declare
their field set; there is no implicit inheritance from a shared base,
and there are **no** root `!{zettel,tag,type,repo}` types — a genre
value (`zettel`, `tag`, `type`, …) is a plain field value, not a
reference to a root type.

A type *may* eventually lift selected fields from its type-of into its
own output (the lightweight substitute for an `!actionable` base type),
but **that mechanism is deferred**: it depends on a maturer
compositional-type / type-of interface and is out of scope here. Genre
does not need it, because each type declares genre directly.

### Genre comes only from the type (id syntax is decoupled)

Object-id syntax stops encoding genre. `ValidateSeqAndGetGenre` is
removed; id shapes (`foo/bar`, `/repo`, `!md`, `@digest`) become naming
conventions for *addressing* objects, not genre classifiers. An object's
genre is read from its projected `_genre` field. This dissolves today's
chicken-and-egg (parse id → derive genre) — the id is just an address,
and genre is data on the object. The `!`-prefix remains the identity
sigil for addressing a type object (`!md`); it simply stops doubling as
a genre selector.

### Projected into the index, with no drift

Genre is projected into `Metadata.Index.Fields` at commit like every
other field — so it is queryable through the existing field path for
free, rather than recomputed on every read as genre is today. There is
no staleness risk: an object's type is pinned by its signature, so the
object (and its projected fields) is immutable until the type is
updated, and a type update forces a **new history entry** that
regenerates the fields. The type-blob-digest carried on each `Field`
(`fields/main.go:54-58`) is the lookup hint that ties a projected value
to the type version that produced it.

### Default and scope queries: one layered model

This is the design point that looked unresolved (repo default
"fallback" vs workspace query "AND"). Inspecting the code shows there is
no real conflict — there are two *separate* mechanisms today that
already coexist:

- **Default genre** — `BuilderOptionDefaultGenres(genres.Zettel)`,
  hardcoded per command (`show.go`), fills the genre of any term that
  omits one (`build_state.go:131-160`, `:366-368`). A *fallback*: an
  explicit genre on a term overrides it.
- **Workspace query** — the workspace config's `query` string
  (`workspace_config_blobs`, `ConfigWithDefaultQueryString`), parsed
  across **all genres** and **AND-intersected** onto every query
  (`builder.go:223-236`, enforced in `main.go:285-287`). A *scope*: it
  cannot be escaped. (Bats `show_workspace_default`: in a workspace
  scoped to `tag-5`, `show :` returns only `tag-5` zettels — default
  genre AND workspace scope, composing fine.)

Once **genre is a field**, both collapse into a single rule:

> A query is the conjunction of the **repo default query**, the
> **workspace default query** (if in a workspace), and the **user
> query**. When two layers constrain the *same single-valued field*
> (genre), the more specific layer wins (user > workspace > repo).
> Constraints on *multi-valued fields* (tags) accumulate (intersect).

The apparent "fallback vs AND" split is really **field cardinality**,
not layering:

- `_genre` is single-valued (one type ⇒ one genre). A user's `_genre=type`
  (or `^_genre=""`) *replaces* the repo's `_genre=zettel` default on that
  field, yielding types (or everything) rather than the empty set. That
  *is* the fallback, derived for free.
- `today` / `tag-5` are multi-valued tag constraints. A workspace scope
  *accumulates* and stays inescapable — exactly today's workspace `query`
  behavior.

Single-valuedness is what licenses override-on-genre; the query engine
needs to know which fields are single-valued (`_genre` is, by construction).

### Unifying workspace and repo config

The workspace config and repo mutable config **already share** the
`Defaults{Type,Tags}` struct (`repo_configs.DefaultsV1` in the repo
config; `DefaultsV1OmitEmpty` embedded in `workspace_config_blobs.V0`),
and at runtime the workspace overlays the repo — **Type overrides, Tags
append** (`env_workspace/main.go:106-126`). The default query is the one
piece that is workspace-only today (`Query` field); the repo's only
"default" is the hardcoded per-command default genre.

The redesign extends that shared overlay to carry the **default query**
and applies the same override/accumulate merge:

- Move the default query out of workspace-only into a shared
  config-overlay both blobs embed, so the **repo** mutable config sets
  the base default query (`_genre=zettel`), replacing the hardcoded
  `BuilderOptionDefaultGenres`.
- A **workspace** overlays it under the single-valued-override /
  multi-valued-accumulate rule above — the exact generalization of
  today's Type-overrides / Tags-append merge, now covering queries.

This is "workspaces are basically repos" (FDR-0005) made concrete at the
config layer: one shared config-overlay (Defaults + default-query), repo
provides the base, workspace overlays.

### Migration: version bump + `import` type-rewrite

Existing objects derive genre from id syntax and carry no `_genre` field;
removing `ValidateSeqAndGetGenre` requires the field. Migration is a
**hard cutover behind a store-version bump** — no permanent id-shape
fallback. The `_genre` field is re-derived once and projected, and the
vehicle is the **`import` command extended with a type-rewriting map**:
import can rewrite an object's type on the way in, and the rewritten /
assigned type is what declares the genre that then projects into the
index. Objects of the null type `!` (no type to declare genre) are
handled by the same type-rewrite map (assign a type) or fall to a
configured default.

### `genres.Genre` kept; index flattened

The `genres.Genre` enum is retained as the set of well-known genre
values — codecs and the genre→default-type map keep their compile-time
dispatch — it simply stops being parsed from id syntax. The `ids.Genre`
bitfield's per-genre **index partitioning is flattened**: genre becomes
a normal indexed field scanned like any other, and the per-genre
`optimizedQueries` dispatch is removed.

## Interface (target)

Genre discrimination becomes a field predicate — no `!`-prefix, no
genre-suffix sigil. (`!md` as an *id* still addresses the md type
object; that is unaffected.)

| Intent | Before | After |
|---|---|---|
| Zettels with a tag (explicit) | `der show tag:z` | `der show _genre=zettel tag` |
| Zettels with a tag (implicit) | `der show tag` | `der show tag` *(repo default query)* |
| Tags with a tag | `der show tag:e` | `der show _genre=tag tag` |
| Types with a tag | `der show [!type tag]:t` | `der show _genre=type tag` |
| All objects with a tag | `der show tag:z,e,t,k` | `der show '^_genre="" tag'` *(provisional)* |

`^_genre=""` is the provisional "any genre" form: it reads as "genre is
not empty" (true for every object) and, being an explicit `_genre`
predicate, overrides the repo default `_genre=zettel` on that
single-valued field. It is unwieldy and overloaded; a terser dedicated
token may replace it.

Because `_genre=…` is an ordinary predicate, the composition the suffix
grammar could not express is native:

```
der show '_genre=zettel | _genre=tag' project   # zettels OR tags, tagged project
der show '_genre=zettel & -archived'             # zettels not archived
der show '^_genre=type'                          # everything that is not a type
```

## Limitations / scope boundaries (intended)

- The `!`-prefix stays as the type-identity sigil in object ids, and the
  id shapes themselves are unchanged — only their role as *genre
  classifiers* is removed.
- This FDR does not add comparison/regex operators to field-match; `=` /
  `^=` are the current surface. Richer operators are a separate concern.
- Field lifting from a type's type-of is out of scope (deferred).

## Open Questions

1. **One layered model vs a separate scope slot.** The layered
   default-query model (repo ∧ workspace ∧ user, single-valued override
   + multi-valued accumulate) appears to subsume *both* today's
   default-genre and workspace-`query` mechanisms. Decide whether a
   distinct always-AND "scope" slot is still wanted (e.g. for a filter a
   user explicitly must not be able to override even on a single-valued
   field), or whether the single model is enough.

2. **Override semantics + single-valued field set.** The precise rule
   for override across a boolean expression (e.g. `_genre` mentioned in
   only one branch of an `|`), and the exact set of single-valued fields
   (`_genre`, and any others). Must reproduce today's per-term
   default-genre behavior.

3. **Config schema + migration.** The shared config-overlay
   (Defaults + default-query) embedded by both `repo_configs` and
   `workspace_config_blobs`; the `repo_configs` and
   `workspace_config_blobs` version bumps; and migrating the existing
   workspace `query` field into the shared slot.

4. **"Any genre" ergonomics.** `^_genre=""` works but is unwieldy and
   overloaded; is a terser dedicated token (e.g. `_genre=*`) worth it,
   and how does it interact with the override rule (Q2)?

5. **(Deferred, non-blocking) field lifting.** How a type lifts fields
   from its type-of — syntax, liftable set, collision rules. Parked
   until the compositional-type / type-of interface matures; genre does
   not depend on it.

## Key Files

| File | Role |
|---|---|
| `go/internal/bravo/ids/types_builtin.go` | `BuiltinType.Genre` — today's per-type genre assertion (builtins only); generalized to a declared field. |
| `go/internal/bravo/ids/main.go` | `ValidateSeqAndGetGenre` (`:258-361`) — removed under full decoupling. |
| `go/internal/alfa/genres/main.go` | Genre enum + bit values — retained as well-known values, no longer id-derived. |
| `go/internal/bravo/ids/genre.go` | `ids.Genre` bitfield — per-genre index partition flattened. |
| `go/internal/juliett/queries/main.go` | `addOptimized` per-genre duplication (`:124-152`); `defaultQuery` AND-application (`:285-287`) — both reworked into the layered model. |
| `go/internal/juliett/queries/build_state.go` | Default-genre fill (`:131-160`, `:366-368`); field-match parse (`:262-279`); genre-suffix parse (`:404-451`) retired. |
| `go/internal/juliett/queries/builder.go` | `defaultQuery` parsed across all genres (`:223-236`) — becomes the layered default-query. |
| `go/internal/juliett/queries/builder_options.go` | `BuilderOptionDefaultGenres` (replaced by config default-query); `builderOptionWorkspace` reading `GetDefaultQueryString` (`:82-86`). |
| `go/internal/juliett/queries/exp_field.go` | `expField` — the composable predicate `_genre=…` reuses. |
| `go/internal/charlie/repo_configs/` | Repo mutable config — gains the shared default-query overlay (base layer). |
| `go/internal/echo/workspace_config_blobs/` | Workspace config — its `Query` field migrates into the shared overlay; overlays the repo default-query. |
| `go/internal/mike/env_workspace/main.go` | Config merge (`:106-126`, Type-override/Tags-append) — extended to merge the default query. |
| `go/internal/alfa/type_blobs/field_definition.go` | `FieldDefinition` — where a type's `_genre` declaration attaches. |
| `go/internal/0/fields/main.go` | `Field` value + `Kind`; the projected `_genre` field's home. |
| `go/internal/oscar/store/field_reader.go` / `field_writer.go` | Field projection at commit; where the type-asserted `_genre` is written. |
| dodder `import` command / `import_plan` | Gains the type-rewriting map that creates genre data during migration. |

## More Information

- **FDR-0005 (Workspace as Repo)** — the basis for unifying the
  workspace and repo config blobs around a shared config-overlay.
- **FDR-0010 (Core Types)** — the null type `!` as the fixed point of
  the recursive parse chain; the `dodder.net` seed-repo model for
  shipping builtin types.
- **`docs/plans/2026-04-06-task-type-genesis-and-haustoria-fields.md`**
  — the fields infrastructure this FDR reuses, and the `!actionable`
  composition idea that the deferred field-lifting mechanism (Q5) would
  eventually replace.
- **`design_patterns-genre_bitfield` skill** — documents the
  enum/bitfield/sigil system this FDR supersedes; will need revision.
  Its sigil table is already stale (`:e`=Tag/etikett, `:t`=Type per
  `genres/main.go:169-223`).
