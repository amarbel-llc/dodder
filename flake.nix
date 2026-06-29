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
      url = "github:amarbel-llc/madder/go/v0.4.0";
      inputs.igloo.follows = "igloo";
      inputs.nixpkgs-master.follows = "nixpkgs-master";
      inputs.utils.follows = "utils";
      inputs.tommy.follows = "tommy";
      inputs.bats.follows = "bats";
      inputs.purse-first.follows = "purse-first";
      inputs.hyphence.follows = "hyphence";
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
      purse-first,
      treelint,
      ...
    }:
    let
      # Burnt into binaries via the fork's auto-injected -ldflags.
      # Single source of truth; `just bump-version` sed-rewrites this line.
      dodderVersion = "0.2.6";
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
    ));
}
