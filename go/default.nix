{
  nixpkgs,
  bats ? null,
  tap ? null,
  tommy,
  madder ? null,
  system,
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
      { batsLaneOutputs = { }; fixtures-current = null; }
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
        # version.bats greps dodderVersion out of the source-of-truth
        # flake.nix; stage it via extraStagedFiles inside the lane.
        flakeNixSrc = ../flake.nix;
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
    inherit pkgs version commit goPkgsTest goFlakeInputs;
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
  # skills (onboarding, usage, zettel IDs, blob stores). The clown plugin
  # protocol disallows `${...}` expansion in stdioServers.command, so the
  # binary path is baked in at build time via Nix substitution: the
  # source-controlled `clown.json.in` uses an `@dodder@` placeholder
  # rewritten to `${dodder}/bin/dodder` here. The `.claude-plugin/plugin.json`
  # template similarly bakes in the dodder version so the manifest can't
  # drift against the binary (mirrors amarbel-llc/madder#204). Consumers
  # wire the plugin into clown by pointing `--plugin-dir` at
  # `${dodder-clown-plugin}/share/purse-first/dodder/`.
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
  '';
in
{
  packages = {
    inherit
      dodder
      dodder-debug
      dodder-debug-race
      dodder-debug-cover
      dodder-clown-plugin
      dodder-go-test
      ;
    default = dodder;
  } // batsLaneOutputs // (
    if fixtures-current == null then { } else { inherit fixtures-current; }
  ) // (
    # Re-surface the input-derived helper derivations as named packages
    # so paved-path agent recipes (test-bats-cover-shim, test-cover-*)
    # can resolve them via `nix build .#<name>` without reaching into
    # the flake inputs by hand. Gated on the same null checks as
    # batsLaneOutputs so non-flake imports stay working.
    if madder == null then { } else { madder-bin = madder.packages.${system}.default; }
  ) // (
    if bats == null then { } else { bats-libs = bats.packages.${system}.bats-libs; }
  );

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
      pandoc
      radicale
      shellcheck
      shfmt
      tree
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
    ];

    GOTOOLCHAIN = "local";
  };
}
