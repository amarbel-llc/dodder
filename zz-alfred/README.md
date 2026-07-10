# zz-alfred

The macOS [Alfred](https://www.alfredapp.com/) workflow for dodder, ported
from the original `zit`-era workflow. It provides keyword and snippet
triggers that shell out to dodder to search, create, edit, and format
zettels.

## Layout

- `workflow/info.plist` — the Alfred workflow definition, and the only
  runtime artifact. Each action's inline `<script>` calls the dodder binary
  directly — there are no wrapper shell scripts. Three build-time placeholders
  are baked into the `<script>`s:
  - `@dodder@` — the dodder binary path, baked to its nix-store path by the
    `dodder-alfred-workflow` Nix package (see `../go/default.nix`).
  - `@repo_id@` — the dodder repo-id the search + edit actions target, baked
    by the home-manager module (see `hm-module.nix`) from the `repoId` option.
  - `@workspace@` — a workspace directory, baked from the `workspace` option;
    used only by the not-yet-migrated `new` actions.

  Search actions inline `'@dodder@' cat-alfred -repo_id '@repo_id@' …` and the
  `edit` action inlines `'@dodder@' edit -repo_id '@repo_id@' -ephemeral …`
  (FDR-0023): dodder resolves the target repo by id (the FDR-0019 scope
  mechanism — no cwd), and for edit spins a throwaway repo-backed workspace
  against it, applies the change, pushes it back, and tears the temp
  workspace down. No `cd`, no persistent `.dodder-workspace` — the actions
  work from anywhere.

  Baking an absolute store path directly into the plist (rather than a
  `run.bash` wrapper that resolved the binary on `PATH`) works because Alfred
  runs each action as a GUI/non-login `/bin/bash -c` with no direnv or login
  `PATH` to lean on. The old `run.bash` existed to hardcode the binary path
  and workspace before nix/direnv; nix baking is the correct replacement, and
  it removes the wrapper indirection entirely.
- `hm-module.nix` — the home-manager module.
- `prune_orphans.py` + `justfile` — tooling used during the port (see
  below).

Search + `edit` use the repo-id path. The remaining write actions (`der new`,
`zn`, Move-to-Dodder) still `cd` into `@workspace@` and run a plain `new`;
migrating them to `-repo_id`/`-ephemeral` needs ephemeral workspace-repos to
inherit the parent's default type (a `new` has no type otherwise) plus more
test coverage (tracked as followups; see also
[dodder#340](https://github.com/amarbel-llc/dodder/issues/340)).

## Triggers

| Keyword / snippet | dodder command |
| --- | --- |
| `der open`, `der open hidden` | `cat-alfred :z,e` / `:?z,e` → `edit` |
| `der new` | `new -edit=true -description ...` |
| `zn` snippet | `new` (one empty zettel) |
| `z` / `zi` snippet | `cat-alfred :z` → `format-blob <id> text` → clipboard |
| `zt` snippet | `cat-alfred :z` |
| `zth` snippet | `cat-alfred ?e` |
| Move to Dodder (file action) | `new -organize -delete` |

## Installing via home-manager

The repo-root flake exposes the module as
`homeManagerModules.dodder-alfred`. It integrates with eng's
`home/alfred.nix` — rather than managing its own prefs symlink, it
contributes the staged workflow to that module's
`programs.alfred.extraWorkflows` option, which `alfred.nix` symlinks into
`Alfred.alfredpreferences/workflows/dodder` on activation. So it must be
composed alongside `programs.alfred`:

```nix
{
  imports = [ dodder.homeManagerModules.dodder-alfred ];

  programs.alfred.enable = true; # provided by eng's home/alfred.nix

  programs.dodder-alfred = {
    enable = true;
    workspace = "/Users/you/dodder-workspace"; # required
  };
}
```

Because it references `programs.alfred.extraWorkflows`, this module is not
standalone: the config must also import `home/alfred.nix` (which defines
that option). Reload Alfred (or restart it) to pick up the workflow.

## Port notes

The original workflow targeted the `zit` binary and had accumulated
TheArchive-era cruft. The port:

- renamed `zit` → `dodder` on every script;
- renamed the `new -bezeichnung` flag to `-description`;
- mapped removed subcommands to current equivalents (`new-empty` → `new`,
  `add -organize -delete` on paths → `new -organize -delete`,
  `format-zettel -mode akte text` → `format-blob <id> text`);
- dropped the `.py` / `.js` / `.jq` helper scripts (unreferenced by
  `info.plist`) and the orphaned icons; and
- dropped two triggers with no current CLI equivalent: the `z add -kind`
  file-copy action (relied on a `z` alias + `up.bash` that were never part
  of the bundle) and the `snippet-address` filter (a removed `.zit/bin`
  helper). See `prune_orphans.py` for the exact removed-object set.

The orphan removal was done once, mechanically, via
`just codemod-prune-plist` (which asserts the plist stays internally
consistent — no dangling connection edges). It is idempotent-guarded and
not part of any recurring build.
