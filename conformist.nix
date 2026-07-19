# dodder's conformist overlay, merged in flake.nix (conformist.lib.evalModule)
# with conformist.lib.presets.eng and the inline tommy TOML-formatter module
# (which needs the `tommy` flake input, so it can't live here). This replaces
# the former hand-written ./conformist.toml: the config is now Nix-generated,
# `nix fmt` runs the generated wrapper, and `just go/check-conformist` builds
# the sandboxed checks.formatting derivation against the same config. See
# conformist-nix(7).
#
# The toml's original job — pinning dodder's formatter set at the git root so
# eng's cwd-aware catch-all wrapper could never silently restyle
# content-addressed or frozen files (amarbel-llc/eng#222) — is now covered
# hermetically instead: the sweatfile hook names (conformist-pre-commit /
# conformist-repair) and the bare `conformist` all resolve to this repo's
# module-generated wrappers on the devShell PATH, which bake this config plus
# the formatter toolchain as store paths.
{ pkgs, lib, ... }:
{
  # Go: goimports (priority 1) runs before gofumpt (priority 2) so gofumpt
  # re-canonicalizes the import-grouped output — the same chain the old
  # conformist.toml / codemod-go-fmt drove.
  programs.goimports.enable = true;
  programs.goimports.priority = 1;
  programs.gofumpt.enable = true;
  programs.gofumpt.priority = 2;
  programs.nixfmt.enable = true;

  # go.mod lives at go/, not the tree root. Without this, goimports/gofumpt
  # run with cwd at the tree root, where Go tooling can't resolve the
  # module — confirmed in langlang (see langlang/conformist.nix) to SILENTLY
  # DELETE correctly-used imports as apparently-unused when the imported
  # package's declared name differs from its path's last segment, because
  # the resolver can't discover which identifier the import provides. That's
  # a silent build break, not a style nit. workingDir (conformist#38) scopes
  # the formatter's cwd to go/, matching dodder's single Go module.
  programs.goimports.workingDir = "go";
  programs.gofumpt.workingDir = "go";

  # shfmt: a raw stanza rather than `programs.shfmt.enable`. The module cannot
  # emit `-ci` (no option for it) and its default includes lack `*.bats`, both
  # of which dodder's project shell style requires: 2-space indent, simplify,
  # case-branch indent, over *.sh / *.bash / *.bats. The frozen snapshot
  # suites under zz-tests_bats/previous_versions/** stay excluded below so
  # migration conformance and fixture hashes are unaffected.
  settings.formatter.shfmt = {
    command = "${pkgs.shfmt}/bin/shfmt";
    options = [
      "-w"
      "-i"
      "2"
      "-s"
      "-ci"
    ];
    includes = [
      "*.sh"
      "*.bash"
      "*.bats"
    ];
  };

  # stylua: raw stanza, deliberately options-free so the output is stylua's
  # defaults (tab indentation) — byte-stable with eng's catch-all AND with the
  # old conformist.toml. The embedded pandoc lua filters are go:embed'd and
  # content-addressed; this lane's output feeds the pinned blob digests in
  # zz-tests_bats/current_version/format_pandoc.bats, so do not add options or
  # a stylua settings file here without regenerating those digests.
  settings.formatter.stylua = {
    command = "${pkgs.stylua}/bin/stylua";
    includes = [ "*.lua" ];
  };

  # rustfmt for the rust/ tree, mirroring the old conformist.toml exactly:
  # skip_children keeps rustfmt from recursing mod files it wasn't handed;
  # edition 2024 matches the crate.
  settings.formatter.rustfmt = {
    command = "${pkgs.rustfmt}/bin/rustfmt";
    options = [
      "--config"
      "skip_children=true"
      "--edition"
      "2024"
    ];
    includes = [ "*.rs" ];
  };

  # eng-versioning(7): dodder's Go module lives in go/, not the tree root,
  # so the linter's default key derivation (root go.mod/Cargo.toml) finds
  # neither and would fail at eval time. Pin the key explicitly instead
  # (dodder#371).
  linters.eng-versioning.key = "DODDER_VERSION";

  # Three presets.eng justfile linters are disabled because bringing dodder's
  # justfile surface into conformance means RENAMING documented user-facing
  # recipes, which is a workflow migration (followup), not a formatting fix:
  #   - justfile-recipe-names / justfile-leaf-noun: `check` and
  #     `generate-seed-types` don't start with a known eng verb (`check` would
  #     become validate/verify-*), and `check` is a bare-verb leaf;
  #   - justfile-task-hierarchy: the seven test-bats-* dev-loop utilities
  #     (tags/targets/race/update-fixtures/update-goldens/snapshot-version/
  #     targets-no-sandbox) are deliberate orphans — several are mutating
  #     regen tools that must NOT run from any aggregate.
  # The other four (justfile-default, justfile-aggregate-comments,
  # justfile-recipe-descriptions, justfile-debug-recipes) stay enabled and
  # pass.
  linters.justfile-recipe-names.enable = lib.mkForce false;
  linters.justfile-leaf-noun.enable = lib.mkForce false;
  linters.justfile-task-hierarchy.enable = lib.mkForce false;

  # NOTE: no shellcheck lane here, mirroring the retired conformist.toml:
  # `just go/check-shellcheck` remains the standalone gate (dodder#323) with
  # its own exclude list (go/vimdiff.bash, the frozen snapshots). Tommy
  # codegen regen likewise stays on the canonical `just build-go-generate` /
  # `just commit-codegen` path — no [linter.tommy-codegen] lane.

  # Excludes layered on conformist's default-excludes (which already cover
  # *.lock — hence flake.lock — plus root-level go.mod/go.sum and LICENSE).
  # conformist compiles excludes via gobwas/glob: a bare literal (no `*`)
  # only matches that exact root-relative path, so dodder's nested go/ files
  # are spelled explicitly.
  settings.excludes = [
    "go/go.sum"
    # [formatter.tommy] formats *.toml; the gomod2nix lock must not be
    # restyled.
    "go/gomod2nix.toml"
    "*.md"
    "result"
    "result-*"
    ".tmp/**"
    ".direnv/**"
    "go/build/**"
    # Frozen fixtures + snapshotted bats suites. Reformatting these would
    # break migration conformance (previous_versions/main.bats) and fixture
    # hashes.
    "zz-tests_bats/previous_versions/**"
  ];
}
