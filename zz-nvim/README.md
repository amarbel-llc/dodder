# dodder.nvim

Neovim tree-sitter grammars and plugin for [dodder](../). Replaces the regex
syntax highlighting in [`zz-vim`](../zz-vim) with real tree-sitter parsers, adds
per-object-type **body-language injection** (nested grammars), and ports
zz-vim's behavioral ftplugin features to lua.

`zz-vim` (classic Vim) and `zz-nvim` (this plugin, neovim-only) are meant to
coexist; pick one per editor.

## What it highlights

Three grammars, each with committed generated parsers (`grammars/*/src/parser.c`)
and `tree-sitter test` corpus tests:

| Grammar           | Parses                                                        | Filetype(s)                     |
| ----------------- | ------------------------------------------------------------ | ------------------------------- |
| `hyphence`        | the object edit format: `---` fenced metadata + a body whose language is injected per object type (see [hyphence(7)](../docs/man.7/hyphence.md)) | `dodder-object`, `dodder-workspace` |
| `doddish`         | the query language `predicate[sigil][genre]` (see [doddish(7)](../docs/man.7/doddish.md)) | (set manually / injected)       |
| `dodder_organize` | organize-text: hyphence header + `#` headings + `-`/`%` box object lines (see [organize-text(7)](../docs/man.7/organize-text.md)) | `dodder-organize`               |

Shared rule modules (`grammars/common/`) parse markl-ids, hyphence metadata
lines, and the box format, and are reused across grammars by `require()`.

## Body-language injection

A hyphence object's body language (markdown, toml, sh, ...) comes from its
type's `vim-syntax-type` field, so it is resolved at runtime — mirroring
zz-vim's `dodder show -format type.vim-syntax-type` shell-out, but non-blocking:

- `queries/hyphence/injections.scm` captures the body and delegates to a custom
  directive `#dodder-injection-language!` (registered in `lua/dodder/injection.lua`).
- The directive reads a per-buffer cache (`b:dodder_body_lang`) with a static
  fallback keyed on the type string, so the first paint has no flicker.
- An async resolver runs the dodder CLI once per buffer, maps the result to a
  tree-sitter parser, caches it, and restarts highlighting so injections re-run.
- `.konfig` and workspace bodies are always TOML. If dodder is not on `PATH`,
  the body falls back to markdown.

## Install

Requires neovim 0.10+ (0.9 works with a jobstart fallback for the async
resolver) and the compiled parsers on the runtimepath.

**Nix** (recommended): the `dodder-nvim` flake package builds the parsers and
stages the plugin:

```
nix build .#dodder-nvim
```

Add its store path to your neovim runtimepath (or via home-manager). It ships
`parser/{hyphence,doddish,dodder_organize}.so`, so neovim's built-in
`vim.treesitter` loads them with no `nvim-treesitter` dependency.

**Manual / development**: build the parsers with the tree-sitter CLI and point
neovim at this directory:

```
for g in hyphence doddish organize; do
  lang=$g; [ "$g" = organize ] && lang=dodder_organize
  (cd grammars/$g && tree-sitter build -o ../../parser/$lang.so)
done
```

then `set rtp^=/path/to/zz-nvim` and `require('dodder').setup()`.

## Setup

`plugin/dodder.lua` calls `require('dodder').setup()` automatically. To
customize, set `vim.g.dodder_no_default_setup = true` and call it yourself:

```lua
require('dodder').setup({
  bin = 'dodder',        -- or vim.g.dodder_bin / $BIN_DODDER
  extensions = {},       -- extra object file extensions
  ftplugin = true,       -- port zz-vim keymaps/equalprg/folding
})
```

The organize filetype has no stable extension (organize temp files use the
configurable `Organize` extension, default `md`), so — like zz-vim — it is
activated programmatically: `:set filetype=dodder-organize`.

Run `:checkhealth dodder` to verify the binary, parsers, and queries.

## Behavioral features (ported from zz-vim)

- `=` formats via `dodder format-object` / `format-organize` (`equalprg`).
- `gf` on a zettel id expands and checks it out, opening it in a new tab.
- `<localleader>z` / `c` / `p` run type-specific actions / copy / preview via
  `vim.ui.select` (copy/preview need the macOS `tacky` / `qlmanage` tools).
- Organize buffers fold by `#` heading depth.

## Testing

```
# grammar corpus tests (needs the tree-sitter CLI)
for g in hyphence doddish organize; do (cd grammars/$g && tree-sitter test); done

# highlight/injection smoke test in neovim
nvim --clean -u tests/minimal_init.lua path/to/object.zettel
# then :InspectTree and :Inspect
```
