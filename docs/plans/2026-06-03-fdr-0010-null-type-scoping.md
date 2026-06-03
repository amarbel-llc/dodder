# FDR-0010 Scoping: Null Type `!` (notation-only) + tool-blob path

**Date:** 2026-06-03 **FDR:** 0010 (Core Types) **Status:** scoping

Scopes the implementation of FDR-0010 starting from the null type `!`.
Supersedes the design assumption in FDR-0010 that the null type must be
*explicit in persisted streams*. See "Key reframing" below.

## Key reframing: the null type already exists as the *empty* type

The central discovery from reading the code: the internal state FDR-0010
calls "the null type" **already exists and already persists** --- it is
the *empty* type, and every persistence layer already round-trips it **by
omission**:

- **Digest / content address** (`echo/object_fmt_digest/format.go:198`):
  `if !metadata.GetType().IsEmpty()` --- the type line is omitted from
  the object hash when the type is empty. An empty-typed object's content
  address is computed as if there were no type field.
- **Box format** (`golf/box_format/transacted.go:272`,
  `checked_out.go:253`): omits the type token for empty-typed objects.
  Omission is unambiguous --- tags never start with `!`.
- **Binary stream index** (`hotel/stream_index/binary_encoder.go:196`,
  `stream_index_fixed/binary_encoder.go:161`): skips the field when
  empty.
- **Commit** (`oscar/store/mutating.go:63`): empty type on a *type-genre*
  object is auto-replaced with the default; for other genres, empty type
  is a legitimate persisted state.

### Consequences

1. **`IsEmpty()` must keep returning `true` for the null type.** The
   digest's stability depends on it. Making `IsEmpty()` return `false`
   (as an early sketch proposed) would force the type line back into the
   hash and re-address *every object that has ever existed*. Non-starter.

2. **No inventory-list v3, no store-version bump, no fixture regen.**
   Persistence keeps the existing omission encoding. The null type is
   **conceptual / notation-only**: it becomes *expressible* (parseable)
   and *displayable*, but the bytes on disk do not change. This removes
   the heaviest part of the FDR's original plan.

3. The representation is **empty `typeStruct.Value`**, unchanged on the
   wire. The only new surface is *notation*: parsing `!` / `!@digest` and
   optionally rendering `!` for display.

## Workstream 1 (revised): make `!` expressible, persistence untouched

The promotion criterion "round-trips through box, hyphence, binary, and
inventory list formats" is **already satisfied for an object's own type**
via omission. The new work concentrates on (a) recognizing bare `!` and
(b) the **blob-reference type lock** path `!@digest`, which is FDR-0010's
actual mechanism (`!md` references grammar/rumdl blobs as `!`-typed).

### 1.1 Type-id parse/recognition --- `bravo/ids/type.go`

- `Set` (:133) trims `.! ` then requires `TagRegex`; bare `!` trims to
  `""` and fails. Add: input that is exactly `!` (after space-trim)
  resolves to the null type (`Value = ""`) without hitting the regex.
  Guard so an unintended empty string is not silently accepted as a type
  except via the explicit `!` spelling.
- Keep `String()` returning `""` for empty (34 callers; digest is
  guarded by `IsEmpty()` so it is safe, but a blanket `!` return risks
  leaking `!` into concatenations). Introduce explicit `!` rendering only
  at the display surfaces that ask for it (1.4).
- `ToSeq` (:195) currently emits `{Op('!'), Identifier("")}`. For the
  null type this should be a single `{Op('!')}` token so matchers and
  round-trips line up.

### 1.2 Tokenizer / genre resolution --- `0/doddish`, `bravo/ids/main.go`

- `doddish/token_matcher.go`: add one-token `TokenMatcherNullType =
  {Op('!')}` and a `TokenMatcherNullTypeLock = {Op('!'), Op('@'),
  Identifier}` (the `!@digest` shape --- no identifier between `!` and
  `@`).
- `ValidateSeqAndGetGenre` (:258): add a bare-`!` case returning
  `genres.Type`. "Recognized without a backing object" falls out for
  free --- genre resolution does no object lookup.

### 1.3 Box reader --- `golf/box_format/read.go` (:209)

- Add cases for bare `!` (object type) and `!@digest` (blob-reference
  lock) alongside the existing `TokenMatcherType` /
  `TokenMatcherTypeLock` cases.

### 1.4 Display rendering (optional, config-gated)

- Box format is **both** the display form *and* the inventory-list
  persistence form (the doddish coder writes box into inventory lists).
  To avoid leaking `!` into persisted streams, any "render `!`
  explicitly" behavior must be a **box-writer option** (default off for
  persistence, on for interactive `show`/checkout) --- this reuses the
  "box writer is configuration-aware, not version-aware" property.
- Open question: do we even need to *display* `!`? For the promotion
  criteria, the only hard requirement is expressing `!@digest` in blob
  reference locks. Displaying a bare `!` on an object's own type line is a
  nicety; defer unless wanted.

### 1.5 Hyphence blob-reference lock --- `echo/object_metadata_fmt_hyphence`

- Parser (`text_parser2.go:188-220`) switches on `TokenMatcherTypeLock`
  / `TokenMatcherType`; add the `!@digest` / bare-`!` cases so a
  `.type`/heredoc can express a null-typed blob reference.
- Writer (`formatter_components.go`): the type-lock string for an empty
  type currently renders `""` and is omitted. Decide whether a
  null-typed blob lock writes `!@digest` (so the reference's type is
  visible/explicit) --- this is the one place explicit `!` is likely
  *wanted*, since the reference type is load-bearing for the graph.

## Workstream 2 --- tool-blob shipping at genesis

The pandoc prototype already proved embed -> blob-store -> genesis +
typed blob references (`romeo/local_working_copy/genesis_pandoc_tools.go`,
`embedded_pandoc_tools.go`). Reuse that path to store the PEG grammar,
rumdl binary, and rumdl config as `!`-typed blobs referenced from `!md`.
Mostly wiring + larger embedded payloads. Phase 1 expedient; Phase 3
(`dodder.net` seed repo) supersedes embedding later.

## Workstream 3 --- langlang + `dodder://` reference extraction

- **langlang is not yet a dependency** (`go.mod` has none; the unrelated
  `0/orgmode_peg` is a separate PEG). Adding it as a Go library triggers
  the **MIT -> GPL-3.0 relicense** (FDR decision #2) --- a real,
  irreversible policy change. Gate on explicit go-ahead.
- `[references]` becomes a **oneof** (`engine` vs `script`); the existing
  script path is `oscar/store/reference_discovery.go`. The langlang
  `engine` slots alongside it.
- **Scheme collision to reconcile:** `dodder://` is **already used** by
  the MCP server with a different grammar (`dodder://objects`,
  `dodder://query/...`, `dodder://types/...`, `dodder://tags/...` in
  `tango/mcp_dodder/resources.go`). FDR-0010 proposes
  `dodder://<yin>/<yang>`, `dodder://!<type>`, `dodder://<tag>`,
  `dodder://<digest>`. The two conventions must be unified before the URI
  scheme is locked.

## Workstream 4 --- rumdl validation at commit

New `[validation]` section (tool + config); run rumdl against the blob at
commit, fail on errors. New hook in the commit path; depends on WS2
shipping the binary.

## Suggested order

1. **WS1** (this doc, revised) --- small now that persistence is
   untouched. Unblocks everything. Ends with `!` / `!@digest` expressible
   and `!`-typed blobs referenceable from `!md`.
2. **WS2** --- reuse pandoc genesis path for grammar/rumdl blobs.
3. **WS3** --- decide relicense + scheme reconciliation *first*, then
   integrate langlang.
4. **WS4** --- rumdl commit-time validation.

## Decisions captured (2026-06-03 session)

- Null type is **conceptual / notation-only**; persistence is not
  changed. No inventory-list v3, no store-version bump, no fixture regen.
- Representation: empty `typeStruct.Value`; `IsEmpty()` stays `true`.
- Start with WS1.
