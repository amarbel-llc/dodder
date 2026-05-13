{
  nixpkgs,
  bats ? null,
  tap ? null,
  tommy,
  madder ? null,
  system,
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

  # Devshell-only test harness for SFTP integration tests. Intentionally
  # NOT included in the `packages` output — release artifacts must not
  # ship a server that accepts any password (mirrors madder/RFC 0001).
  dodder-test-sftp-server = pkgs.buildGoApplication {
    pname = "dodder-test-sftp-server";
    version = "0.0.0";
    src = ./.;
    pwd = ./.;
    subPackages = [ "cmd/dodder-test-sftp-server" ];
    modules = ./gomod2nix.toml;
    go = pkgs.go_1_26;
    GOTOOLCHAIN = "local";
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
        inherit pkgs batsSrc dodder-test-sftp-server;
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
    inherit pkgs version commit;
  };

  dodder-go-test = goTestAttrs.dodder-go-test;

  # dodder-debug: same source as `dodder` but compiled with `-tags
  # debug`. The debug tag enables debug-only subcommands
  # (debug-print-probe-index, etc.) and runtime pool-repool
  # poisoning (see CLAUDE.md). It's what `just build` puts in
  # `go/build/debug/` and what the existing batman bats path
  # exercises. The bats nix lanes use this binary so test assertions
  # that enumerate subcommands (complete_subcmd) match what the
  # dev-shell path sees.
  #
  # Devshell-only: NOT in the `packages` output — release artifacts
  # must not ship the debug instrumentation.
  dodder-debug = pkgs.buildGoApplication {
    pname = "dodder-debug";
    inherit version commit;
    src = ./.;
    pwd = ./.;
    subPackages = [
      "cmd/der"
      "cmd/dodder"
    ];
    modules = ./gomod2nix.toml;
    go = pkgs.go_1_26;
    GOTOOLCHAIN = "local";
    tags = [ "debug" ];
  };

  dodder = pkgs.buildGoApplication {
    pname = "dodder";
    inherit version commit;
    src = ./.;
    pwd = ./.;
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
in
{
  packages = {
    inherit dodder dodder-go-test;
    default = dodder;
  } // batsLaneOutputs // (
    if fixtures-current == null then { } else { inherit fixtures-current; }
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
      dodder-test-sftp-server
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
