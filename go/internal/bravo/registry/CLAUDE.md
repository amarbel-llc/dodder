# registry

Per-host, best-effort index of every dodder repo created on this machine
(RFC-0007 registry v1). Feeds `dodder repos-list`; ADVISORY ONLY — never
consulted for repo resolution (FDR-0019 owns that).

Twin of madder's `go/internal/bravo/registry` — same primitive with
`Utility = "dodder"`. Both copies are stdlib-only and utility-agnostic,
pending extraction into dewey; keep them synchronized.

## Contract

- Index dir: `$XDG_STATE_HOME/dodder/index/` — `$XDG_STATE_HOME` is read
  directly from the environment (no env_dir scope machinery, no walk-up),
  so XDG-redirecting sandboxes isolate the index for free.
- Entry: symlink named `hex(sha256(Clean(repo-dir)))[:8]` (16 hex chars,
  no extension) → absolute path of the repo's `config-seed`.
- Writes are TOCTOU-safe (symlink to `.tmp-<pid>-<nano>`, then rename).
- Dangling symlink = stale entry, classified, never an error; pruned by
  `GC` after a retention window (default 30d, `<=0` no-op, no tombstones).

## Key Functions

- `Register(baseAbsPath, configAbsPath)`: best-effort registration
- `Entries()`: all entries, live + dangling, sorted by key
- `GC(retention)`: prune aged dangling entries
- `Key(baseAbsPath)`: recompute an entry's filename (for dedup)
