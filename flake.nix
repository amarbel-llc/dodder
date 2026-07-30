{
  inputs = {
    igloo.url = "https://code.linenisgreat.com/igloo/archive/master.tar.gz";
    nixpkgs-master.url = "github:NixOS/nixpkgs/567a49d1913ce81ac6e9582e3553dd90a955875f";
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";

    bats = {
      url = "https://code.linenisgreat.com/bats/archive/master.tar.gz";
      inputs.igloo.follows = "igloo";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
    };

    tap = {
      url = "https://code.linenisgreat.com/tap/archive/master.tar.gz";
      inputs.igloo.follows = "igloo";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
      inputs.bats.follows = "bats";
    };

    purse-first = {
      url = "https://code.linenisgreat.com/purse-first/archive/master.tar.gz";
      inputs.igloo.follows = "igloo";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
    };

    # hyphence format library, extracted from madder (madder #253). madder
    # consumes it too; madder.inputs.hyphence.follows below keeps both on the
    # same rev (RFC 0001 flake-input-go_mod bridge — see go/gomod.nix).
    hyphence = {
      url = "https://code.linenisgreat.com/hyphence/archive/master.tar.gz";
      inputs.igloo.follows = "igloo";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
      inputs.bats.follows = "bats";
      inputs.purse-first.follows = "purse-first";
    };

    madder = {
      # tracks madder master; move it with `just go/update-flake-input madder`
      # (updates flake.lock + go.mod + gomod2nix in one shot) rather than
      # pinning a rev here.
      url = "https://code.linenisgreat.com/madder/archive/master.tar.gz";
      inputs.igloo.follows = "igloo";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
      inputs.tommy.follows = "tommy";
      inputs.bats.follows = "bats";
      inputs.purse-first.follows = "purse-first";
      inputs.hyphence.follows = "hyphence";
      inputs.piggy.follows = "piggy";
    };

    # The markl-id framework home (piggy#183 ownership inversion),
    # sourced via goFlakeInputs so a piggy bump only touches flake.lock
    # — no go.mod / gomod2nix.toml lockstep edits. Its go-pkgs producer
    # is scoped to go/ (module code.linenisgreat.com/piggy/go, no
    # subPath). madder consumes it too; madder.inputs.piggy.follows
    # above keeps both on the same rev.
    piggy = {
      url = "https://code.linenisgreat.com/piggy/archive/master.tar.gz";
      inputs.igloo.follows = "igloo";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
      inputs.bats.follows = "bats";
      inputs.purse-first.follows = "purse-first";
    };

    tommy = {
      url = "https://code.linenisgreat.com/tommy/archive/master.tar.gz";
      inputs.igloo.follows = "igloo";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
      inputs.bats.follows = "bats";
      inputs.tap.follows = "tap";
    };

    # conformist: the linter + formatter multiplexer (treefmt successor).
    # Config is Nix-generated from ./conformist.nix (+ presets.eng) via
    # conformist.lib.evalModule — see conformistEval below and
    # conformist-nix(7). Replaces the retired treelint input (a stale
    # pre-adoption placeholder) and the hand-written conformist.toml.
    conformist = {
      url = "https://code.linenisgreat.com/conformist/archive/master.tar.gz";
      inputs.igloo.follows = "igloo";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
    };
    tap.inputs.treefmt-nix.follows = "igloo/treefmt-nix";
    utils.inputs.systems.follows = "igloo/systems";
    igloo.inputs.nixpkgs-master.follows = "nixpkgs-master";
    tap.inputs.gomod2nix.follows = "purse-first/gomod2nix";
    tap.inputs.purse-first.follows = "purse-first";
    madder.inputs.tap.follows = "tap";
    madder.inputs.conformist.follows = "conformist";
    piggy.inputs.conformist.follows = "conformist";
    purse-first.inputs.conformist.follows = "conformist";
    tommy.inputs.conformist.follows = "conformist";
    madder.inputs.doppelgang.follows = "hyphence/doppelgang";
    hyphence.inputs.conformist.follows = "conformist";
    bats.inputs.conformist.follows = "conformist";
    # hyphence's langlang subtree (rust guest-filter tooling) brings its own
    # tap -> crane/rust-overlay chain, colliding with dodder's own tap input
    # and doubling the lock (crane_2/rust-overlay_2). Collapse onto dodder's
    # tap, mirroring madder's exact deep-follows shape (madder flake.nix).
    hyphence.inputs.langlang.inputs.tap.inputs.crane.follows = "tap/crane";
    hyphence.inputs.langlang.inputs.tap.inputs.rust-overlay.follows = "tap/rust-overlay";
    hyphence.inputs.piggy.inputs.jcardsim.follows = "piggy/jcardsim";
    hyphence.inputs.piggy.inputs.oracle-javacard-sdks.follows = "piggy/oracle-javacard-sdks";
    hyphence.inputs.piggy.inputs.pivapplet.follows = "piggy/pivapplet";
    # piggy's own langlang pin (piggy#183's markl-id framework pulls in the
    # same langlang subtree hyphence does) is bit-identical to hyphence's —
    # collapse onto hyphence's copy rather than deep-following piggy's
    # separately, mirroring madder's exact fix (madder flake.nix, go-module
    # rename playbook wave 2 / piggy+madder leg).
    piggy.inputs.langlang.follows = "hyphence/langlang";
    # dodder#378: expose langlang as `inputs.langlang` (consumed below as
    # the `packages.langlang` output -- see outputs.packages, `--inputs-from`
    # does not work for a follows-based indirect input like this one,
    # confirmed empirically). Reuses the ALREADY collapsed hyphence/piggy
    # node above rather than declaring an independent `url`-based input,
    # which would reintroduce the exact tap/crane/rust-overlay diamond
    # the two lines above this exist to collapse.
    langlang.follows = "hyphence/langlang";
  };

  outputs =
    {
      self,
      igloo,
      utils,
      bats,
      tap,
      tommy,
      madder,
      hyphence,
      piggy,
      purse-first,
      conformist,
      langlang,
      ...
    }:
    let
      # eng-versioning(7): version.env at repo root is the single source of
      # truth for the release version, burnt into binaries via the fork's
      # auto-injected -ldflags. dodder's Go module lives in go/, so
      # buildGoApplication's pwd is scoped there — its version.env auto-read
      # (gomod2nix(7) VERSION AND COMMIT INJECTION) can't reach a repo-root
      # file. Read it here instead and pass the value through explicitly,
      # mirroring madder's flake.nix (same sub-directory-module shape).
      # `just bump-version` sed-rewrites version.env, not this file.
      dodderVersion = builtins.head (
        builtins.match ".*DODDER_VERSION=([^\n]+).*" (builtins.readFile ./version.env)
      );
      dodderCommit = self.shortRev or self.dirtyShortRev or "unknown";
    in
    (utils.lib.eachDefaultSystem (
      system:
      let
        # Needed for the mkGoPkgs producer call in go/gomod.nix.
        # buildGoApplication / mkGoEnv consumers live in go/default.nix.
        pkgs = import igloo { inherit system; };

        gomod = import ./go/gomod.nix {
          inherit
            pkgs
            system
            madder
            hyphence
            piggy
            tap
            tommy
            purse-first
            ;
          # Scope the producer at go/ so downstream consumers reference
          # go-pkgs directly with no subPath. Dodder's repo root has
          # no Go-relevant assets, so a full-repo filter would only
          # bloat the closure.
          src = self + "/go";
        };

        inherit (gomod.goPkgs) go-pkgs go-pkgs-test;

        # The tommy TOML formatter lane ([formatter.tommy], `tommy fmt` over
        # *.toml). Inlined here rather than in ./conformist.nix because it
        # needs the `tommy` flake input, which a standalone module file can't
        # see (same shape as madder's flake.nix). getExe' with an explicit
        # binary name: tommy lacks meta.mainProgram.
        conformistTommyModule =
          { ... }:
          {
            settings.formatter.tommy = {
              command = pkgs.lib.getExe' tommy.packages.${system}.default "tommy";
              options = [ "fmt" ];
              includes = [ "*.toml" ];
            };
          };

        # conformist config, Nix-generated from ./conformist.nix merged with
        # the eng-convention preset (conformist.lib.presets.eng). Backs
        # `nix fmt` (build.wrapper) and the read-only formatting gate
        # (checks.formatting = build.check, run by `just go/check-conformist`
        # via the root `lint` recipe). Replaces the retired treelint-fmt
        # wrapper + hand-written conformist.toml. See conformist-nix(7).
        conformistEval = conformist.lib.evalModule pkgs {
          imports = [
            conformist.lib.presets.eng
            conformistTommyModule
            ./conformist.nix
          ];
          package = conformist.packages.${system}.default;
        };

        # Hook eval: the pre-commit / merge-repair wrappers. EXPLICITLY
        # formatters-only (no presets.eng): the eng convention linters gate at
        # `just lint`, not at authoring time (mirrors madder's split). The
        # wrappers bake the generated config + the formatter toolchain as
        # store paths and land on the devShell PATH as `conformist-pre-commit`
        # / `conformist-repair`, so the sweatfile hook names inherited from
        # eng resolve to THIS repo's formatter set — which is what retires
        # conformist.toml's eng#222 pinning role (eng's cwd-aware catch-all
        # wrapper can no longer be the one that runs here).
        conformistHooksEval = conformist.lib.evalModule pkgs {
          imports = [
            conformistTommyModule
            ./conformist.nix
          ];
          package = conformist.packages.${system}.default;
        };

        result = import ./go/default.nix {
          nixpkgs = igloo;
          inherit
            bats
            tap
            tommy
            madder
            system
            ;
          # Module-generated conformist wrappers for the devShell PATH: the
          # config-baked `conformist` (manual runs inside the devShell shadow
          # eng's cwd-aware catch-all wrapper) plus the two sweatfile hook
          # names. See conformistEval / conformistHooksEval above.
          conformistWrapper = conformistEval.config.build.wrapper;
          conformistPreCommit = conformistHooksEval.config.build.preCommit;
          conformistRepair = conformistHooksEval.config.build.repair;
          # Pivot self-consumption onto the published artifact: every
          # buildGoApplication in go/default.nix uses this as `src`,
          # so the same closure downstream consumers receive via
          # go-pkgs-test is what dodder builds itself from. Contract
          # test for the producer-side split — if the filter ever
          # drops a file the build needs, this build breaks (#217).
          goPkgsTest = go-pkgs-test;
          inherit (gomod) goFlakeInputs;
          version = dodderVersion;
          commit = dodderCommit;
          man7Src = ./docs/man.7;
          batsSrc = ./zz-tests_bats;
        };
      in
      {
        packages = result.packages // {
          inherit go-pkgs go-pkgs-test;
          # Dogfood the hook wrappers: `nix build .#conformist-pre-commit`
          # forces the hook-eval output to build and is the same wrapper the
          # devShell puts on PATH (mirrors madder).
          conformist-pre-commit = conformistHooksEval.config.build.preCommit;
          conformist-repair = conformistHooksEval.config.build.repair;
          # dodder#378: re-exposed as a real dodder package (NOT resolved
          # via `nix build --inputs-from`, which the check-go-repool/
          # -seqerror/-defererr recipes use for purse-first#foo -- that
          # mechanism does not work for `langlang`, since it's a
          # FOLLOWS-based (indirect) top-level input, not a direct one;
          # `--inputs-from` failed with "cannot find flake 'flake:langlang'
          # in the flake registries" against the collapsed node, confirmed
          # empirically this session). `go/justfile`'s check-grammar /
          # test-grammar-vectors resolve this directly instead:
          # `nix build --no-link --print-out-paths '../..#langlang'`.
          langlang = langlang.packages.${system}.default;
        };
        checks = result.checks // {
          # Sandboxed read-only formatting gate: `conformist check` against a
          # /nix/store snapshot of the tracked tree. Built by
          # `just go/check-conformist` (root `lint` lane).
          formatting = conformistEval.config.build.check self;
        };
        devShells.default = result.devShells.default;
        # `nix fmt` runs the generated conformist wrapper (see conformistEval).
        formatter = conformistEval.config.build.wrapper;
      }
    ))
    // {
      # System-independent home-manager module that installs the macOS
      # Alfred workflow (zz-alfred). It closes over `self` to resolve the
      # per-system `dodder-alfred-workflow` package, so consumers only
      # need to set `programs.dodder-alfred.workspace`. See
      # zz-alfred/hm-module.nix.
      homeManagerModules.dodder-alfred = import ./zz-alfred/hm-module.nix self;
    };
}
