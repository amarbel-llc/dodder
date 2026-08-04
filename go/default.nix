{
  nixpkgs,
  bats ? null,
  tap ? null,
  tommy,
  madder ? null,
  system,
  # Module-generated conformist wrappers (flake.nix conformistEval /
  # conformistHooksEval): the config-baked `conformist` plus the
  # `conformist-pre-commit` / `conformist-repair` sweatfile hook wrappers.
  # All three go on the devShell PATH. Null on the non-flake
  # `import ./go/default.nix` path — the devShell then simply omits them.
  conformistWrapper ? null,
  conformistPreCommit ? null,
  conformistRepair ? null,
  # Filtered Go source tree (test-superset shape) produced by
  # mkGoPkgs in go/gomod.nix and threaded through flake.nix. Every
  # buildGoApplication self-consumes this as `src`/`pwd` so dodder
  # builds itself from the same artifact downstream consumers see —
  # contract test for the go-pkgs / go-pkgs-test split (#217).
  # Defaulted to ./. so non-flake callers (`import ./go/default.nix`
  # without flake context) still work — they just build from the
  # live worktree.
  goPkgsTest ? ./.,
  # Flake-input bridge table (see ./gomod.nix). Defaulted to {} so
  # non-flake callers degrade to organic gomod2nix.toml resolution
  # for cross-amarbel deps (#218).
  goFlakeInputs ? { },
  man7Src ? null,
  # Source dir for the bats integration suite. Required by the
  # `batsLaneOutputs` block; null disables it (so direct
  # `import ./go/default.nix` callers without a flake context still
  # work — they just don't get the bats-* lane outputs).
  batsSrc ? null,
  # Passed to buildGoApplication's `version` and `commit` attrs; the
  # fork's nixpkgs auto-injects them as `-X main.version` and
  # `-X main.commit` ldflags on every subPackage. Defaulted so direct
  # `import ./go/default.nix` (outside the flake) still works; release
  # builds always override via flake.nix.
  version ? "dev",
  commit ? "unknown",
}:
let
  pkgs = import nixpkgs {
    inherit system;
  };

  # bats integration test lanes + hermetic fixture generator. Auto-
  # discovered file_tags become `bats-${tag}` flake outputs;
  # `bats-default` runs everything; `fixtures-current` produces the
  # `.dodder/.madder/.fixtures.env` payload for the active store
  # version. Empty when batsSrc/bats/madder are absent (non-flake
  # import path), so direct `import ./go/default.nix` callers without
  # a flake context stay working — they just don't get the bats-* or
  # fixtures-* outputs.
  batsAttrs =
    if batsSrc == null || bats == null || madder == null then
      {
        batsLaneOutputs = { };
        fixtures-current = null;
      }
    else
      import ./bats.nix {
        inherit pkgs batsSrc;
        # Lanes run against the debug-tagged binary to match what the
        # existing batman test path exercises (subcommand surface,
        # repool poisoning, etc.). The release `dodder` package stays
        # untagged and ships via packages.default.
        dodder = dodder-debug;
        # batsLane + bats-libs both come from amarbel-llc/bats so the
        # lane builder and its helper-lib path move together.
        batsLane = bats.lib.${system}.batsLane;
        bats-libs = bats.packages.${system}.bats-libs;
        madder-bin = madder.packages.${system}.default;
        # SFTP test server for haustoria_orgmode bats lanes. Sourced
        # from madder's flake (amarbel-llc/madder#177) rather than
        # rebuilt locally — see haustoria_orgmode.bats for the
        # consumption site.
        madder-test-sftp-server-bin = madder.packages.${system}.madder-test-sftp-server;
        # version.bats greps DODDER_VERSION out of the source-of-truth
        # version.env; stage it via extraStagedFiles inside the lane.
        versionEnvSrc = ../version.env;
        # show.bats's pandoc-discover tests reference a worktree-root
        # script directory that lives outside zz-tests_bats/.
        pandocRefsSrc = ../zz-pandoc-refs;
      };

  batsLaneOutputs = batsAttrs.batsLaneOutputs;
  fixtures-current = batsAttrs.fixtures-current;

  # Sandboxed Go unit test lane. Runs `go test -tags test,debug ./...`
  # inside a nix-build sandbox so tests cannot leak into the developer's
  # real $HOME / XDG dirs (see go/go-tests.nix for the motivating bug).
  # Unconditional — its only flake-input dependency is nixpkgs, which is
  # always available.
  goTestAttrs = import ./go-tests.nix {
    inherit
      pkgs
      version
      commit
      goPkgsTest
      goFlakeInputs
      ;
  };

  dodder-go-test = goTestAttrs.dodder-go-test;

  # dodder-debug: same source as `dodder` but compiled with `-tags
  # debug`. The debug tag enables debug-only subcommands
  # (debug-print-probe-index, etc.) and runtime pool-repool poisoning
  # (see CLAUDE.md). The bats nix lanes use this binary so test
  # assertions that enumerate subcommands (complete_subcmd) match
  # what the dev-shell path sees. Surfaced as a flake output so
  # dev-loop recipes (test-bats-targets, explore-*, etc.) resolve it
  # via `nix build --no-link --print-out-paths .#dodder-debug`.
  dodder-debug = pkgs.buildGoApplication {
    pname = "dodder-debug";
    inherit version commit goFlakeInputs;
    src = goPkgsTest;
    pwd = goPkgsTest;
    subPackages = [
      "cmd/der"
      "cmd/dodder"
    ];
    modules = ./gomod2nix.toml;
    go = pkgs.go_1_26;
    GOTOOLCHAIN = "local";
    tags = [ "debug" ];
  };

  # Race-instrumented variant of dodder-debug. The fork's
  # pkgs.buildGoRace also runs `go test -race ./...` as a checkPhase,
  # which needs the test build tag the dev-shell uses. Bats injects
  # this binary externally and the dodder-go-test derivation already
  # covers unit tests, so we just append -race to buildFlagsArray
  # ourselves (mirroring the cover variant below) and skip the test
  # phase.
  dodder-debug-race = dodder-debug.overrideAttrs (old: {
    pname = "${old.pname}-race";
    CGO_ENABLED = 1;
    preBuild = (old.preBuild or "") + ''
      buildFlagsArray+=("-race")
    '';
  });

  # Coverage-instrumented variant of dodder-debug. Bats runs externally
  # (it writes its own GOCOVERDIR), so the build only needs the -cover
  # flags — pkgs.buildGoCover's in-derivation coverIntegrationCommand
  # is unnecessary. preBuild appends to buildFlagsArray, the bash array
  # the fork's go-config-hook splats into the go install command.
  dodder-debug-cover = dodder-debug.overrideAttrs (old: {
    pname = "${old.pname}-cover";
    preBuild = (old.preBuild or "") + ''
      buildFlagsArray+=("-cover" "-covermode=atomic")
    '';
  });

  # Non-stripped variant of dodder-debug for use with delve. nix's
  # fixup phase strips .debug_* (DWARF) sections from the standard
  # debug binary, which `dlv` needs for breakpoints and stack traces;
  # dontStrip retains them. Investigation/explore aid (see the
  # explore-trace-sftp-init justfile recipe); not shipped.
  dodder-debug-dwarf = dodder-debug.overrideAttrs (old: {
    pname = "${old.pname}-dwarf";
    dontStrip = true;
  });

  dodder = pkgs.buildGoApplication {
    pname = "dodder";
    inherit version commit goFlakeInputs;
    src = goPkgsTest;
    pwd = goPkgsTest;
    subPackages = [
      "cmd/der"
      "cmd/dodder"
      "cmd/dodder-gen_man"
    ];
    modules = ./gomod2nix.toml;
    go = pkgs.go_1_26;
    GOTOOLCHAIN = "local";

    nativeBuildInputs = pkgs.lib.optionals (man7Src != null) [
      pkgs.pandoc
    ];

    postInstall = ''
      mkdir -p $out/share/man/man1
      $out/bin/dodder-gen_man $out/share/man/man1
      rm $out/bin/dodder-gen_man
    ''
    + pkgs.lib.optionalString (man7Src != null) ''
      mkdir -p $out/share/man/man7
      for f in ${man7Src}/*.md; do
        name="$(basename "$f" .md)"
        pandoc -s -t man "$f" -o "$out/share/man/man7/$name.7"
        # .ss 12 0 = disable double sentence spacing
        # .na = ragged-right (no justification)
        ${pkgs.gnused}/bin/sed -i '3a\.\\" Formatting overrides\n.ss 12 0\n.na' "$out/share/man/man7/$name.7"
      done
    '';
  };

  # dodder-clown-plugin stages a clown plugin (see clown-plugin-protocol(7)
  # / clown-json(5)) that exposes dodder's MCP server and ships user-facing
  # skills (onboarding, usage, zettel IDs; blob stores are documented in
  # blob-store(7), not a skill). The clown plugin
  # protocol disallows `${...}` expansion in stdioServers.command, so the
  # binary path is baked in at build time via Nix substitution: the
  # source-controlled `clown.json.in` uses an `@dodder@` placeholder
  # rewritten to `${dodder}/bin/dodder` here. The `.claude-plugin/plugin.json`
  # template similarly bakes in the dodder version so the manifest can't
  # drift against the binary (mirrors amarbel-llc/madder#204). Consumers
  # wire the plugin into clown by pointing `--plugin-dir` at
  # `${dodder-clown-plugin}/share/purse-first/dodder/`.
  #
  # hooks/ ships a PreToolUse hook that auto-approves the read-only MCP
  # tools (#244); clown auto-discovers `$CLAUDE_PLUGIN_ROOT/hooks/hooks.json`
  # the same way it does for the spinclass plugin. The decision table
  # lives in `dodder hook` (internal/bravo/claude_hooks, unit-tested);
  # the handler script gets the binary path baked in like clown.json.
  dodder-clown-plugin = pkgs.runCommand "dodder-clown-plugin" { } ''
    pluginRoot=$out/share/purse-first/dodder
    mkdir -p $pluginRoot/.claude-plugin
    cp -r ${../plugins/dodder/skills} $pluginRoot/skills
    substitute \
      ${../plugins/dodder/.claude-plugin/plugin.json.in} \
      $pluginRoot/.claude-plugin/plugin.json \
      --replace-fail '@version@' '${version}'
    substitute \
      ${../plugins/dodder/clown.json.in} \
      $pluginRoot/clown.json \
      --replace-fail '@dodder@' '${dodder}/bin/dodder'
    mkdir -p $pluginRoot/hooks
    ${pkgs.jq}/bin/jq -e . ${../plugins/dodder/hooks/hooks.json} > /dev/null
    install -m 0644 ${../plugins/dodder/hooks/hooks.json} $pluginRoot/hooks/hooks.json
    substitute \
      ${../plugins/dodder/hooks/handler} \
      $pluginRoot/hooks/handler \
      --replace-fail '@dodder@' '${dodder}/bin/dodder'
    chmod 0755 $pluginRoot/hooks/handler
  '';

  # dodder-vim packages the editor support tree at ../zz-vim (ftdetect,
  # ftplugin, syntax, autoload for dodder object/workspace filetypes) as
  # a standard Vim/Neovim plugin, so consumers stop pointing their editor
  # config at a source checkout (#245). It is a plain copy — no build
  # step — but buildVimPlugin gives it the conventional pack layout,
  # helptags, and a store path other flakes / home-manager modules can
  # depend on. The path is referenced directly (like the clown plugin),
  # so the non-flake `import ./go/default.nix` path builds it too.
  dodder-vim = pkgs.vimUtils.buildVimPlugin {
    pname = "dodder-vim";
    version = version;
    src = ../zz-vim;
    meta = {
      description = "Vim/Neovim filetype detection, syntax, and ftplugin support for dodder objects, types, tags, configs, and workspaces";
    };
  };

  # dodder-nvim packages the neovim tree-sitter plugin at ../zz-nvim. Unlike
  # dodder-vim (a plain copy), the three grammars must be compiled: each is
  # built with pkgs.tree-sitter.buildGrammar from its committed src/parser.c
  # (generate = false, so the shared ../common/*.js rule modules are not needed
  # at build time), then the resulting parsers are dropped into $out/parser/ as
  # <lang>.so so neovim's built-in vim.treesitter auto-loads them off the
  # runtimepath -- no nvim-treesitter dependency. lua/, plugin/, and queries/
  # ride along from the buildVimPlugin copy. Referenced directly like dodder-vim
  # so the non-flake `import ./go/default.nix` path builds it too. dodder-vim is
  # kept for legacy Vim users; dodder-nvim is neovim-only.
  dodder-nvim-grammars =
    builtins.mapAttrs
      (
        lang: subdir:
        pkgs.tree-sitter.buildGrammar {
          language = lang;
          version = version;
          src = ../zz-nvim/grammars/${subdir};
          generate = false;
        }
      )
      {
        hyphence = "hyphence";
        doddish = "doddish";
        dodder_organize = "organize";
      };

  dodder-nvim = pkgs.vimUtils.buildVimPlugin {
    pname = "dodder-nvim";
    version = version;
    src = ../zz-nvim;
    postInstall = ''
      mkdir -p "$out/parser"
    ''
    + pkgs.lib.concatStrings (
      pkgs.lib.mapAttrsToList (
        lang: drv: ''cp ${drv}/parser "$out/parser/${lang}.so"'' + "\n"
      ) dodder-nvim-grammars
    );
    meta = {
      description = "Neovim tree-sitter grammars (hyphence, doddish, organize) and plugin for dodder objects, queries, and organize buffers, with per-type body-language injection";
    };
  };

  # dodder-alfred-editor-wrapper is the EDITOR the workflow's edit/new
  # actions bake in via the `@editor@` placeholder (substituted per-user in
  # ../zz-alfred/hm-module.nix, not here — see dodder-alfred-workflow below).
  # Alfred script actions run headless with no controlling TTY, but dodder's
  # editor integration (go/lib/alfa/editor) never spawns a terminal itself —
  # it just execs $EDITOR against the process's own stdio. Without this
  # wrapper, `der new` / `der open`'s edit action exec nvim/vim headless,
  # which fails to open a UI and exits silently (no visible symptom at all).
  # This wrapper detects the no-TTY case and spawns kitty running nvim/vim
  # instead, reconstructing behavior the old eng home/local-bin/editor
  # script used to provide before its removal broke this. kitty is
  # deliberately NOT a runtimeInput: this derivation is evaluated on every
  # eachDefaultSystem platform (including Linux, where the workflow is never
  # consumed), so kitty is resolved from PATH or the macOS .app bundle at
  # runtime instead, exactly like the original script.
  dodder-alfred-editor-wrapper = pkgs.writeShellApplication {
    name = "dodder-alfred-editor-wrapper";
    runtimeInputs = [ pkgs.neovim ];
    text = builtins.readFile ../zz-alfred/workflow/editor-wrapper.bash;
  };

  # dodder-alfred-workflow stages the macOS Alfred workflow at ../zz-alfred.
  # Each Alfred action's inline `<script>` calls the dodder binary directly
  # (no wrapper scripts for the dodder invocation itself): the `@dodder@`
  # placeholder is baked to the binary's nix-store path at build time here,
  # mirroring how dodder-clown-plugin bakes `@dodder@` into clown.json.
  # Read/search actions inline `cd @workspace@ && @dodder@ …`; the edit
  # action inlines `@dodder@ edit -ephemeral -parent @workspace@ …`
  # (FDR-0023). The `@workspace@` and `@editor@` placeholders are
  # intentionally left intact here: they are per-user/per-consumer config,
  # substituted later by the home-manager module (../zz-alfred/hm-module.nix)
  # from the `workspace` and `editorPackage` options respectively — the
  # latter defaults to dodder-alfred-editor-wrapper above but can be
  # overridden. Baking into the plist (rather than a run.bash wrapper) works
  # because Alfred runs each action as GUI/non-login `/bin/bash -c` with no
  # direnv/PATH to lean on, so an absolute store path is exactly what's
  # needed. The path is referenced directly, so the non-flake
  # `import ./go/default.nix` path builds it too. Consumers point Alfred (or
  # the home-manager module) at `${dodder-alfred-workflow}/share/dodder/alfred/workflow`.
  dodder-alfred-workflow = pkgs.runCommand "dodder-alfred-workflow" { } ''
    workflowRoot=$out/share/dodder/alfred/workflow
    mkdir -p "$workflowRoot"
    substitute \
      ${../zz-alfred/workflow/info.plist} \
      "$workflowRoot/info.plist" \
      --replace-fail '@dodder@' '${dodder}/bin/dodder'
  '';

in
{
  packages = {
    inherit
      dodder
      dodder-debug
      dodder-debug-race
      dodder-debug-cover
      dodder-debug-dwarf
      dodder-clown-plugin
      dodder-vim
      dodder-nvim
      dodder-alfred-workflow
      dodder-alfred-editor-wrapper
      dodder-go-test
      ;
    default = dodder;
  }
  // batsLaneOutputs
  // (if fixtures-current == null then { } else { inherit fixtures-current; })
  // (
    # Re-surface the input-derived helper derivations as named packages
    # so paved-path agent recipes (test-bats-cover-shim, test-cover-*)
    # can resolve them via `nix build .#<name>` without reaching into
    # the flake inputs by hand. Gated on the same null checks as
    # batsLaneOutputs so non-flake imports stay working.
    if madder == null then
      { }
    else
      {
        madder-bin = madder.packages.${system}.default;
        # SFTP test harness (amarbel-llc/madder#177), re-surfaced the
        # same way as madder-bin so it's buildable as
        # `nix build .#madder-test-sftp-server` without hardcoding the
        # pinned madder rev anywhere (batman-path bats recipes don't
        # get go/bats.nix's netCapExtraBinaries wiring, so they need
        # an explicit build + MADDER_TEST_SFTP_SERVER export instead).
        madder-test-sftp-server = madder.packages.${system}.madder-test-sftp-server;
      }
  )
  // (if bats == null then { } else { bats-libs = bats.packages.${system}.bats-libs; });

  # NB: no `formatter` attr here anymore — `nix fmt` is wired directly in
  # flake.nix to the conformist module eval's build.wrapper.

  # Wired into the flake's `checks.<system>.*` so `nix flake check` runs
  # the sandboxed Go unit-test lane. Additional checks (e.g. the bats
  # lanes) can drop in here without further flake.nix changes.
  checks = {
    go-test = dodder-go-test;
  };

  docker = pkgs.dockerTools.buildImage {
    name = "dodder";
    tag = "latest";
    copyToRoot = [ dodder ];
    config = {
      Cmd = [ "${dodder}/bin/dodder" ];
      Env = [ ];
      ExposedPorts = {
        "9000/tcp" = { };
      };
    };
  };

  devShells.default = pkgs.mkShell {
    packages = [
      pkgs.gomod2nix
      tommy.packages.${system}.default
    ]
    ++ (with pkgs; [
      bash-language-server
      delve
      fish
      gnumake
      go_1_26
      gofumpt
      golangci-lint
      golines
      gopls
      gotools
      govulncheck
      gum
      httpie
      just
      lsof
      nixfmt
      # nodejs + tree-sitter: `tree-sitter generate`/`test` for the zz-nvim
      # grammars (hyphence, doddish, organize) shells out to node to
      # evaluate grammar.js.
      nodejs
      pandoc
      radicale
      shellcheck
      shfmt
      tree
      tree-sitter
      yq-go
    ])
    ++ pkgs.lib.optionals (bats != null) [
      bats.packages.${system}.default
    ]
    ++ pkgs.lib.optionals (tap != null) [
      tap.packages.${system}.tap-dancer
    ]
    ++ pkgs.lib.optionals (madder != null) [
      madder.packages.${system}.default
    ]
    # The module-generated conformist wrappers (see the parameter docs at the
    # top of this file). `conformist` here shadows eng's home-profile
    # cwd-aware wrapper inside the devShell, so a manual `conformist` run
    # always uses dodder's own config — the eng#222 protection the retired
    # conformist.toml used to provide by its mere presence at the git root.
    ++ pkgs.lib.optionals (conformistWrapper != null) [ conformistWrapper ]
    ++ pkgs.lib.optionals (conformistPreCommit != null) [ conformistPreCommit ]
    ++ pkgs.lib.optionals (conformistRepair != null) [ conformistRepair ];

    GOTOOLCHAIN = "local";
  };
}
