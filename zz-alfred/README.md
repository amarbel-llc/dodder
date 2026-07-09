# zz-alfred

The macOS [Alfred](https://www.alfredapp.com/) workflow for dodder, ported
from the original `zit`-era workflow. It provides keyword and snippet
triggers that shell out to dodder to search, create, edit, and format
zettels.

## Layout

- `workflow/info.plist` — the Alfred workflow definition. Every object
  that runs a script shells out to `./run.bash <dodder-subcommand>`.
- `workflow/run.bash` — entry point. `cd`s into the dodder workspace, then
  execs the dodder binary. Contains two build-time placeholders:
  - `@dodder@` — the dodder binary path, baked in by the
    `dodder-alfred-workflow` Nix package (see `../go/default.nix`).
  - `@workspace@` — the workspace directory, baked in by the home-manager
    module (see `hm-module.nix`) from the required `workspace` option.

  Left unsubstituted (e.g. importing the raw workflow by hand), both fall
  back to the `DODDER_BIN` / `DODDER_WORKSPACE` env vars, then to `dodder`
  on `PATH` and `$PWD`.
- `hm-module.nix` — the home-manager module.
- `prune_orphans.py` + `justfile` — tooling used during the port (see
  below).

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
