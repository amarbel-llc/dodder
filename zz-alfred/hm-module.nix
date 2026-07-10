# home-manager module that installs the dodder Alfred workflow.
#
# Exposed from the repo-root flake as `homeManagerModules.dodder-alfred`
# (see ../flake.nix). It closes over the flake's `self` so it can resolve
# the per-system `dodder-alfred-workflow` package built in ../go/default.nix
# without the consumer having to thread it in.
#
# Integration: this module does NOT symlink into Alfred's prefs directory
# itself. It contributes the staged workflow to `programs.alfred.extraWorkflows`
# (the attrsOf-path option that eng's home/alfred.nix defines precisely so
# other modules can add workflows). home/alfred.nix owns the prefs symlink
# and, on activation, ln -sfn's each extraWorkflows entry into
# Alfred.alfredpreferences/workflows/<name>. So enabling this module beside
# `programs.alfred` is all that's needed — no prefs-path plumbing here.
#
# The workflow package already baked the dodder binary path into
# info.plist's `@dodder@` placeholder. This module supplies the remaining
# per-user config: `@repo_id@` (from the required `repoId` option — the repo
# every action targets via -repo_id; search + edit read it, and the new/zn/
# Move-to-Dodder write actions create against it via -ephemeral).
self:
{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.programs.dodder-alfred;

  workflowPkg = self.packages.${pkgs.system}.dodder-alfred-workflow;

  # The staged workflow: take the package's bundle (which already baked
  # @dodder@ into info.plist) and substitute the per-user @repo_id@ that every
  # action targets via -repo_id (search + edit read it; the write actions
  # create against it via -ephemeral). A runCommand over the store path yields a
  # plain directory that alfred.nix's activation symlinks into workflows/dodder.
  stagedWorkflow = pkgs.runCommand "dodder-alfred-workflow-configured" { } ''
    mkdir -p "$out"
    substitute \
      "${workflowPkg}/share/dodder/alfred/workflow/info.plist" \
      "$out/info.plist" \
      --replace-fail '@repo_id@' '${cfg.repoId}'
  '';
in
{
  options.programs.dodder-alfred = {
    enable = lib.mkEnableOption "the dodder Alfred workflow";

    repoId = lib.mkOption {
      type = lib.types.str;
      example = "work";
      description = ''
        The dodder repo-id every action targets via -repo_id (the FDR-0019
        scope mechanism — the same targeting show/cat-alfred/the MCP use).
        Typically an XDG-user repo name like "work", or "default". Search and
        edit read from it; the new/zn/Move-to-Dodder write actions create
        against it via -ephemeral (a temp repo-backed workspace pulled from this
        repo, pushed back, torn down). No cwd is used, so the actions work from
        anywhere.
      '';
    };

    workflowName = lib.mkOption {
      type = lib.types.str;
      default = "dodder";
      description = ''
        The attribute/directory name the workflow is registered under in
        `programs.alfred.extraWorkflows` (and thus the name it appears as
        under Alfred.alfredpreferences/workflows/). Rarely needs changing.
      '';
    };
  };

  # Contribute the staged workflow to the shared extraWorkflows option that
  # eng's home/alfred.nix owns. mkIf keeps this inert unless the module is
  # enabled; the consumer must also have `programs.alfred.enable = true`
  # for alfred.nix to actually perform the symlink.
  config = lib.mkIf cfg.enable {
    programs.alfred.extraWorkflows.${cfg.workflowName} = stagedWorkflow;
  };
}
