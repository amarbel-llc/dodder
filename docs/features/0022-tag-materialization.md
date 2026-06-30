---
status: exploration
date: 2026-06-30
issue: https://github.com/amarbel-llc/dodder/issues/306
decision: pursue config-gated auto-materialization (option C); implementation
  tracked as a follow-up. This document records current behavior and the
  decision; it does not change the materialization code path.
---

# Tag materialization

## Problem statement

Applying a tag to an object stores only a tag **string** on that object's
metadata. A tag **object** (genre tag, type `!toml-tag-v1`) is created only when
explicitly authored — `new -object-id <tag>` or an `organize` heading. Nothing
on the commit path materializes a tag object from a tag string.

This surprises users of a tag-heavy repo that never hand-authored tag objects:

- `query [":e"]` (all tag objects) returns nothing.
- `query-tag` returns empty — it searches only tag objects.
- The startup banner reports `Indexed: N type(s), 0 tag(s)`.
- Meta-tag laddering (an object matching a tag filter through that tag's
  meta-tags) cannot apply, because there is no tag object to hang meta-tags on.

Meanwhile `query ["<tag>"]` (filter objects by a bare tag name) works fine — it
matches the tag string on each object, materialized or not. So the surprise is
specifically in the tag-*discovery* surface (`:e`, `query-tag`) and in meta-tag
features, not in tag *filtering*.

## Current behavior (as of this writing)

| Operation | Sees string-only tags? | Sees materialized tag objects? |
|---|---|---|
| `query ["<tag>"]` (filter objects) | yes | yes |
| `query [":e"]` / `query-tag` (discover tags) | no | yes |
| meta-tag laddering | no | yes |

Code references:

- **Tag strings, not objects, on commit.** `delta/objects` `AddTagString` /
  `AddTagPtr` store the string and the per-object tag-path index only; the
  checkin/commit path creates no tag object.
- **Types DO auto-materialize (the asymmetry).** `oscar/store/mutating.go`
  `addMissingTypes` → `addTypeIfNecessary` → `createType` auto-creates a missing
  *type* object on commit, with an opt-out (`CommitOptions.DontAddMissingType`,
  used by remote transfer so a pull does not invent types). There is **no
  `addMissingTags`** analog today.
- **History / flip-flop.** An `addMissingTags` did exist previously and was
  removed: auto-materializing every applied tag made the store messy (object and
  index churn, and typos became permanent tag entities). Explicit-only is the
  current *intentional* state, not an oversight.
- **"Realizing" tags is a different thing.** `oscar/store/dormancy_and_tags.go`
  `applyDormantAndRealizeTags` computes the tag-path / implicit-tag closure
  against config at commit/read time. That is not object materialization.
- **Discovery surfaces read tag objects only.** `mcp_dodder` `query-tag` →
  `tagIdx.query` (built from `show :e`); `countUniqueTags` (the banner) counts
  tag objects.
- **Meta-tags require tag objects.** `november/store_config` `recompileTags`
  builds the implicit-tag map only from authored tag objects (`config.Tags`);
  `GetImplicitTags` returns empty for a string-only tag. So meta-tag enrichment
  and laddering fundamentally need the tag object to exist.

## The pivotal question: can `query-tag` work without materialization?

Partly.

- **Listing** tags-in-use without materialization is *possible* but needs new
  infrastructure: nothing today enumerates the distinct tag strings applied
  across objects. It would require either a query-time scan of all objects or a
  new persistent tag-string index maintained on commit.
- **Meta-tags cannot** be served without tag objects — they are compiled from
  authored tag objects. A `query-tag` over string-only tags would return bare
  names with empty `tags` (meta-tag) fields.

So a "works without materialization" `query-tag` is a *degraded* `query-tag`
(names, no meta-tags), and it still costs us a new index — roughly the same
machinery that materialization needs, with less of the payoff.

## Options

- **A. Explicit-only + tag-string listing.** Keep tags string-only; add a
  tag-string index/scan so `query-tag` and `:e` can list tags-in-use. Keeps the
  store clean. Downsides: meta-tags only for hand-authored tags (laddering stays
  partial), and a second tag enumeration surface to maintain.
- **B. Re-introduce auto-materialization.** Re-add `addMissingTags` mirroring
  `addMissingTypes`. `query-tag`, `:e`, and meta-tag laddering all "just work."
  Downsides: the churn / typo-permanence that got it removed before.
- **C. Config-gated auto-materialization.** Add a repo-config toggle (default
  off = today's behavior); when on, commit materializes missing tag objects.
  Both worlds available per repo. Cost: a config field plus both code paths and
  their tests.

## Decision

**Pursue option C — config-gated auto-materialization — with the gate default
off.** Rationale:

- `query-tag`'s real value is meta-tags and laddering, which require tag objects.
  A string-only index (option A) is a half-measure that still builds new
  enumeration infrastructure.
- The original objection (store messiness) is mitigated as the tag type matures
  (filtered `!toml-tag-v1` tags, the #307 fix, the #309 typefulness cleanup), so
  re-introduction is lower-risk than when it was removed.
- The config gate (default off) preserves today's behavior exactly, keeps the
  re-introduction reversible, and lets a repo opt in where tag discovery and
  meta-tag laddering matter.

The implementation is **out of scope for this document** and is tracked as a
follow-up: re-add `addMissingTags` behind a `CommitOptions` / repo-config flag,
recompile tags so newly materialized tags gain their meta-tags, add bats
coverage, and consider a one-time backfill/migration for existing string-only
tags. Until then, behavior is unchanged.

## Immediate companion change (shipped with this document)

`query-tag` is made **non-silent**: an empty result now returns a hint that it
searches only materialized tag objects, that string tags will not appear, and
that `query ["<tag>"]` filters objects by a tag string regardless of
materialization. This removes the "silent null in a tag-full repo" ambiguity
without waiting on the materialization decision. The current behavior is also
documented in `doddish(7)` (the tag-name predicate and the `:e` example).

## Related

- #306 (this exploration), #307 (materialized filterless tag query, fixed),
  #309 (make `tag_blobs.Blob` typeful).
- `addMissingTypes` in `oscar/store/mutating.go` is the model the
  re-introduced `addMissingTags` should mirror.
