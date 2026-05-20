---
name: features-zettel_ids
description: Use when interacting with dodder ZettelIds (two-part IDs like `ceroplastes/midtown`), seeding a new repo via `dodder init -yin` / `-yang`, choosing predictable or random ID allocation, or interpreting "zettel ids exhausted" errors at the command line.
---

# ZettelId System

Dodder assigns each zettel a unique two-part string identifier (e.g.
`ceroplastes/midtown`). Each repo has a finite pool of available IDs, fixed
at `dodder init` time; once that pool is empty, no more zettels can be
created in that repo.

## ID Format

A ZettelId has the form `left/right`, where both halves are non-empty
identifiers. The left and right halves are drawn from two word lists called
**Yin** (left side) and **Yang** (right side). The same word list pair is
used for the lifetime of the repo.

## Seeding the ID Pool

At `dodder init` time the Yin and Yang word lists are copied from files
passed via the `-yin` and `-yang` flags. The total ID space for the repo
equals `len(Yin) * len(Yang)` --- two 200-word lists give 40 000 possible
zettel IDs.

```bash
dodder init \
  -yin path/to/yin.txt \
  -yang path/to/yang.txt \
  ...
```

The lists are immutable after init: once a repo is created, you cannot
swap in a different Yin/Yang to enlarge the pool. Pick lists large enough
for the lifetime of the repo (or accept that you may need to spin up a new
repo when you run out).

## Allocation Modes

Two modes, controlled by a CLI config flag:

- **Random (default).** Each new zettel gets an unpredictable ID from the
  remaining pool. Preferred for human use --- gives zettels memorable,
  non-sequential names.
- **Predictable.** Always picks the next ID in deterministic order. Used by
  tests and tooling that want stable output.

## Exhaustion

When the pool is empty, `dodder new` (and anything else that mints zettels)
fails with `zettel ids exhausted`. There is no in-place expansion path ---
remediation is repo-level (start a new repo with a larger Yin/Yang, or
archive existing zettels into a different store).
