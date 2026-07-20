# zz-alfred

The macOS [Alfred](https://www.alfredapp.com/) workflow for dodder, ported
from the original `zit`-era workflow. It provides keyword and snippet
triggers that shell out to dodder to search, create, edit, and format
zettels.

## Layout

- `workflow/info.plist` — the Alfred workflow definition, and the only
  runtime artifact. Each action's inline `<script>` calls the dodder binary
  directly — there are no wrapper shell scripts. Two build-time placeholders
  are baked into the `<script>`s:
  - `@dodder@` — the dodder binary path, baked to its nix-store path by the
    `dodder-alfred-workflow` Nix package (see `../go/default.nix`).
  - `@repo_id@` — the dodder repo-id every action targets, baked by the
    home-manager module (see `hm-module.nix`) from the `repoId` option.

  Search actions inline `'@dodder@' cat-alfred -repo_id '@repo_id@' …`; the
  `edit` action inlines `'@dodder@' edit -repo_id '@repo_id@' -ephemeral …`;
  and the write actions (`der new`, `zn`, Move-to-Dodder) inline
  `'@dodder@' new -repo_id '@repo_id@' -ephemeral …` (FDR-0023). dodder
  resolves the target repo by id (the FDR-0019 scope mechanism — no cwd), and
  for edit/new spins a throwaway repo-backed workspace against it (inheriting
  the parent's default type), applies the change, pushes it back, and tears the
  temp workspace down. No `cd`, no persistent `.dodder-workspace` — the actions
  work from anywhere.

  Baking an absolute store path directly into the plist (rather than a
  `run.bash` wrapper that resolved the binary on `PATH`) works because Alfred
  runs each action as a GUI/non-login `/bin/bash -c` with no direnv or login
  `PATH` to lean on. The old `run.bash` existed to hardcode the binary path
  and a workspace before nix/direnv; nix baking + `-repo_id`/`-ephemeral` is
  the correct replacement, and it removes the wrapper indirection entirely.
- `hm-module.nix` — the home-manager module.
- `prune_orphans.py` + `justfile` — tooling used during the port (see
  below).

Every action now targets the repo by id: search + `edit` read from it, and the
write actions (`der new`, `zn`, Move-to-Dodder) create against it via
`-ephemeral` (see also
[dodder#340](https://code.linenisgreat.com/dodder/issues/340)).

## Triggers

| Keyword / snippet | dodder command |
| --- | --- |
| `der open`, `der open hidden` | `cat-alfred :z,e` / `:?z,e` → `edit -ephemeral` |
| `der new` | `new -ephemeral -edit=true -description ...` |
| `zn` snippet | `new -ephemeral -edit=true` (one zettel) |
| `z` / `zi` snippet | `cat-alfred :z` → `format-blob <id> text` → clipboard |
| `zt` snippet | `cat-alfred :z` |
| `zth` snippet | `cat-alfred ?e` |
| Move to Dodder (file action) | `new -ephemeral -organize -delete` |

All commands run with `-repo_id <repoId>`.

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
    repoId = "work"; # required — the repo every action targets via -repo_id
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
