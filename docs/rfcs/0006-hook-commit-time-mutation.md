---
status: draft
date: 2026-06-30
---

# Hook Commit-Time Mutation

## Abstract

Dodder runs user-authored lua hooks during the commit pipeline (`on_pre_commit`,
`on_commit_fields`, `on_new`) and during formatting (`on_format`). Today a hook
may mutate only an object's **tags** (and id); the lua↔object binding writes
nothing else back. To support field-driven behavior — the motivating case being
the built-in actionable types, where a *done* recurring `!chore`/`!habit` should
advance its `due` field and reset `status` to `todo` — a hook must be able to
read and mutate **typed fields**, which are canonically stored in the object's
blob. Doing this naively (a hook triggering a fresh commit of the mutated
object) re-enters the commit pipeline and re-fires the hooks, risking an
unbounded cycle. This RFC specifies a bounded, **in-band** model: a single
defined injection point at which a commit-time hook may mutate the *running*
commit's fields and tags, after which the pipeline performs exactly one
write-back pass that does **not** re-invoke hooks. It defines, normatively, what
a commit-time hook MAY and MUST NOT mutate, and the guarantee that a hook's own
mutations cannot re-trigger it.

## Introduction

A type blob carries a `hooks` lua string (`WithStringLuaHooks`). During a commit,
`store.tryPrecommit` runs, in order:

1. `SaveBlob` — the daughter's blob is stored and its digest computed.
2. `tryPreCommitHooks` → lua `on_pre_commit(kinder, mutter)` — the kinder table
   exposes genre/id/type + **tags**; it does **not** expose fields, and at this
   point the field index is not yet populated.
3. `tryWriteFields` — the fields-writer script projects daughter field *edits*
   into the blob (`DODDER_FIELD_*` → blob), then the blob is re-saved.
4. `tryReadFields` — the fields-reader script projects the blob into
   `Metadata.Index.Fields` (status/urgency/priority/due/recurrence).
5. `tryPostFieldHooks` → lua `on_commit_fields(kinder, mutter)` — added by the
   actionable-types work; the kinder table now also exposes the projected
   **fields** (read-only) via `kinder.Fields`. A hook may mutate tags here, and
   `applyDormantAndRealizeTags` (which runs later, after `tryPrecommit`) makes a
   tag-driven dormancy change take effect. This is how the built-in archive
   behavior (`done` `!task` / `cancelled` → `zz-archive`) is implemented.
6. `tryNewHook` → lua `on_new` (genesis-only).

The binding `ToLuaTableV1` projects tags + (now) read-only fields into the
`kinder` table; `FromLuaTableV1` reads back only **tags** and id. Fields, type,
description, and the blob are explicit non-goals in the binding today (`// TODO`
markers). So a hook **cannot** change a field value or the blob: it can decide
*whether* to archive (a tag) but cannot *advance a due date* (a field).

The actionable recurrence behavior needs exactly that field mutation. The
spike that gated this work (see `docs/plans/2026-06-29-merge-actionable-types.md`,
Part B) confirmed the gap and rejected two naive paths:

- An **out-of-band re-commit** — a hook computes the advanced object and commits
  it — re-enters `tryPrecommit` and re-fires `on_commit_fields`, which can loop
  unless every hook is perfectly self-terminating. Unbounded by construction.
- **Reordering or doubling** existing stages changes the semantics of all
  existing hooks.

This RFC specifies the bounded alternative.

## Requirements Language

The key words MUST, MUST NOT, SHOULD, MAY are to be interpreted as in RFC 2119.

## The injection point

`on_commit_fields` (stage 5 above) is designated **the** commit-time injection
point for field-aware mutation. It already runs after the fields are projected
(so they are readable) and before the object is finalized. No new stage is
introduced; existing `on_pre_commit` hooks (stage 2, tags-only, pre-field) are
unaffected and retain their current semantics.

Within `on_commit_fields`, a hook receives:

- `kinder.Fields.<name>` — the projected field values (readable).
- `kinder.Etiketten` — the explicit tags (readable, writable).
- `kinder.Gattung` / `kinder.Kennung` / `kinder.Typ` — genre / id / type
  (readable).

A hook MAY:

- add or remove **tags** (already supported);
- assign new values to `kinder.Fields.<name>` to mutate a **typed field**
  (new capability specified here).

A hook MUST NOT:

- change `kinder.Kennung` (object id) or `kinder.Gattung` (genre) from within
  `on_commit_fields` — identity is fixed for the running commit;
- initiate a nested commit, checkout, or any operation that re-enters the
  commit pipeline (see Cycle prevention);
- rely on observing another object's mutation within the same commit batch
  (each object's hook sees only its own `kinder`/`mutter`).

Whether `Typ` (retyping) is mutable here is an open question (see below); this
RFC does not permit it.

## Write-back model (the bounded pass)

**Scope (Phase 1).** This RFC's initial scope is **field write-back only**: a
hook mutates field values (and tags), and the pipeline rewrites the blob *from*
those fields. A hook may NOT write the blob directly, and may NOT retype the
object, at this time. Both are flagged as possible future expansions (see Future
work); the present design is deliberately the minimal capability that unblocks
the actionable recurrence behavior.

After `on_commit_fields` returns, the pipeline applies the hook's mutations to
the running commit in a single bounded pass:

1. `FromLuaTableV1` reads back the mutated **tags** (as today) **and** the
   mutated **field values** from `kinder.Fields` into the daughter's
   `Metadata.Index.Fields` (new binding capability).
2. If and only if any field changed, the pipeline runs **one** write-back: the
   fields-writer projects the new field values into the blob (`DODDER_FIELD_*`
   → blob), the blob is re-saved (new digest), and the fields-reader re-projects
   the blob into `Index.Fields` so the index and blob agree.
3. This write-back pass MUST NOT invoke any hook.

The write-back reuses the existing fields-writer / fields-reader machinery
(`tryWriteFields` / `tryReadFields`), which already round-trips field values
through the blob; it is run a second time, gated on the hook having changed a
field, and with hooks suppressed.

The number of write-back passes per commit is exactly **one**, independent of
how many fields the hook changed. A hook does not get to observe the result of
its own write-back and mutate again within the same commit.

## Cycle prevention

The cycle guarantee rests on three properties, all normative:

1. **Single invocation.** Each commit-time hook function is invoked at most once
   per object per commit.
2. **Hook-free write-back.** The post-hook write-back pass (re-write + re-save +
   re-read) runs with hooks suppressed, so a hook's field mutation cannot
   re-fire the hook.
3. **No re-entry.** A hook MUST NOT trigger a nested commit. The implementation
   SHOULD enforce this with a re-entrancy guard (a per-store/per-context flag
   set for the duration of the commit) that fails loudly if a commit is
   initiated while one is in progress, rather than silently recursing.

As defense-in-depth, hooks SHOULD be **idempotent**: applying the hook to an
object that already reflects the hook's effect SHOULD be a no-op. For the
actionable recurrence hook this is natural — it only advances `due` when
`status == "done"` and simultaneously resets `status` to `todo`, so a re-run
sees `todo` and does nothing.

## Capability matrix (normative)

For a hook running at `on_commit_fields`:

| dimension | read | write |
|---|---|---|
| genre (`Gattung`) | yes | no |
| id (`Kennung`) | yes | no |
| type (`Typ`) | yes | no (not at this time; possible future feature) |
| tags (`Etiketten`) | yes | yes |
| fields (`Fields`) | yes | **yes (Phase 1)** |
| blob | no (only via fields) | Phase 1: derivative only (rewritten from fields); direct write is future |
| description (`Bezeichnung`) | no | no (future) |

In Phase 1 a hook never sets the blob directly; the blob is rewritten by the
pipeline from the hook's field mutations. This avoids exposing raw blob bytes /
digest manipulation to lua and keeps the blob and field index consistent by
construction. A future revision MAY grant hooks direct blob mutation under a
strict ordering rule (see Future work).

## The motivating consumer

With this model, the built-in actionable recurrence hook becomes:

```lua
on_commit_fields = function(kinder, mutter)
  local f = kinder.Fields
  if not f then return end
  if f.status == "done" and f.recurrence ~= nil and f.recurrence ~= "" then
    f.due = advance(f.due, f.recurrence)  -- date + ISO-8601 duration
    f.status = "todo"
  end
  -- archive cases (cancelled, done !task) continue to set tags as today
end
```

`advance` (date arithmetic over an ISO-8601 duration) is provided to the hook
environment (a host-exposed lua function), since gopher-lua has no date library;
its exact surface is an implementation concern, not part of this RFC's wire
contract.

## Future work: hook blob mutation

Phase 1 lets a hook mutate fields (and tags) and rewrites the blob *from* those
fields; it does not let a hook write the blob directly. A future revision MAY
add direct blob mutation, motivated by hooks whose effect is not expressible as
field edits (e.g. normalizing or templating the blob body). Two ordering models
are the candidates; exactly one MUST be chosen when this is specified:

- **Sequential (fields → blob).** Field mutations are applied first and manifest
  a new blob (as in Phase 1); a subsequent hook-supplied blob write is then
  applied on top of that blob. The ordering is strict and one-directional so the
  result is deterministic and a fields-vs-blob conflict has a defined winner
  (the blob write, applied last).
- **Mutually exclusive (fields XOR blob).** A given hook invocation may write
  *either* fields *or* the blob, but not both, eliminating any fields-vs-blob
  ordering ambiguity entirely.

Both preserve the single-pass, hook-free write-back and the cycle guarantees of
this RFC; they differ only in whether field and blob mutation may coexist in one
hook invocation. The choice is deferred until a concrete consumer needs direct
blob mutation.

## Alternatives considered

- **Dedicated sweep command** — a `dodder` command that queries `done` recurring
  actionables and advances them out-of-band. Lower-risk (no commit-path
  mutation) and viable, but explicit-invocation (manual/cron) and splits the
  behavior engine across a hook (archive) and a command (recurrence). This RFC
  enables the *automatic-on-commit* path; the command remains a valid complement
  or fallback.
- **Fields-writer-only transition** — encode recurrence advancement in the yq
  fields-writer script. The writer runs before fields are read and cannot read
  `status` to gate the transition, cannot add a tag, and would push nontrivial
  date arithmetic into yq/shell. Rejected.
- **Out-of-band re-commit** — rejected for the cycle reason above.

## Migration and compatibility

- Existing hooks are unaffected: `on_pre_commit` keeps its tags-only, pre-field
  semantics; `on_commit_fields` gains a *write* capability for fields that no
  existing hook uses.
- The binding change (`FromLuaTableV1` field write-back) and the gated write-back
  pass are additive. Types without an `on_commit_fields` hook, or hooks that
  don't mutate fields, pay nothing beyond the existing single field round-trip.
- The capability is exercised first by the opt-in built-in actionable types
  (`IncludeBuiltinActionableTypes`).

## Open questions

- **Type mutation.** Should `on_commit_fields` be allowed to retype an object
  (e.g. a state machine that promotes `!task` → `!archived-task`)? This RFC
  forbids it pending a separate analysis of re-typing mid-commit.
- **Write-back mechanism.** Reusing `tryWriteFields` (which diffs daughter vs
  mother field values) for the post-hook pass needs care: the hook's mutations
  are on the daughter's `Index.Fields`, and the writer must pick them up. The
  alternative is a dedicated hook-driven blob rewrite. To be settled in the
  implementation plan.
- **Re-entrancy guard granularity.** Per-store flag vs per-commit context vs a
  bounded depth counter — the strongest enforceable guarantee with the least
  intrusion into the commit path.
- **Error semantics.** If the write-back fails (e.g. the writer script errors on
  the hook's new field value), is the whole commit rejected, or does it fall
  back to the pre-hook state? (Lean: rejected, consistent with
  `IgnoreHookErrors`.)
- **Multi-object batches.** Confirm that per-object single-invocation holds
  across an import/batch commit and that one object's write-back cannot perturb
  another's hook.

## References

- FDR 0017 — type-defined field index (the field model these hooks read/write).
- RFC 0001 — hyphence format (the blob the fields round-trip through).
- `docs/plans/2026-06-29-merge-actionable-types-design.md` — the actionable
  types and the behavior engine this RFC unblocks.
- The Part B spike (lua hook capability gate) in
  `docs/plans/2026-06-29-merge-actionable-types.md`.
