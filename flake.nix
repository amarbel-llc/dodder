{
  inputs = {
    igloo.url = "github:amarbel-llc/igloo";
    nixpkgs-master.url = "github:NixOS/nixpkgs/567a49d1913ce81ac6e9582e3553dd90a955875f";
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";

    bats = {
      url = "github:amarbel-llc/bats";
      inputs.igloo.follows = "igloo";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
    };

    tap = {
      url = "github:amarbel-llc/tap";
      inputs.igloo.follows = "igloo";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
      inputs.bats.follows = "bats";
    };

    purse-first = {
      url = "github:amarbel-llc/purse-first";
      inputs.igloo.follows = "igloo";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
    };

    # hyphence format library, extracted from madder (madder #253). madder
    # consumes it too; madder.inputs.hyphence.follows below keeps both on the
    # same rev (RFC 0001 flake-input-go_mod bridge — see go/gomod.nix).
    hyphence = {
      url = "github:amarbel-llc/hyphence/go/v0.2.0";
      inputs.igloo.follows = "igloo";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
      inputs.bats.follows = "bats";
      inputs.purse-first.follows = "purse-first";
    };

    madder = {
      # pinned past go/v0.4.0 for the piggy markl cutover (madder#255:
      # markl core lives in piggy's go module, madder's registrations
      # are madder-only, dodder registers its own dodder-* purposes);
      # repoint to the next go/vX tag when one lands.
      url = "github:amarbel-llc/madder/0063d397ab4004e68b00ab0e8a4bbc5a457072f0";
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
    # is scoped to go/ (module github.com/amarbel-llc/piggy/go, no
    # subPath). madder consumes it too; madder.inputs.piggy.follows
    # above keeps both on the same rev.
    piggy = {
      url = "github:amarbel-llc/piggy";
      inputs.igloo.follows = "igloo";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
      inputs.bats.follows = "bats";
      inputs.purse-first.follows = "purse-first";
    };

    tommy = {
      url = "github:amarbel-llc/tommy";
      inputs.igloo.follows = "igloo";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
      inputs.bats.follows = "bats";
      inputs.tap.follows = "tap";
    };

    treelint = {
      url = "github:amarbel-llc/treelint";
      inputs.igloo.follows = "igloo";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
    };
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
      treelint,
      ...
    }:
    let
      # Burnt into binaries via the fork's auto-injected -ldflags.
      # Single source of truth; `just bump-version` sed-rewrites this line.
      dodderVersion = "0.2.13";
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

        result = import ./go/default.nix {
          nixpkgs = igloo;
          inherit
            bats
            tap
            tommy
            madder
            treelint
            system
            ;
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
        };
        inherit (result) checks;
        devShells.default = result.devShells.default;
        formatter = result.formatter;
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
